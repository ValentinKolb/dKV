package server

import (
	"context"
	"fmt"
	"github.com/ValentinKolb/dKV/lib/db"
	"github.com/ValentinKolb/dKV/lib/db/engines/maple"
	"github.com/ValentinKolb/dKV/lib/store"
	"github.com/ValentinKolb/dKV/lib/store/dstore"
	"github.com/ValentinKolb/dKV/lib/store/lstore"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/serializer"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/puzpuzpuz/xsync/v3"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
	"time"

	_ "net/http/pprof"
)

var Logger = logger.GetLogger("rpc")

// serverShard is a struct that represents a shard in the RPC server
// It contains the shard ID, the store it encapsulates and the adapter
// that handles requests for the store
type serverShard struct {
	Store   store.IStore
	Adapter IRPCServerAdapter
}

// NewRPCServer creates a new RPC server
// It takes a config, transport and serializer as parameters
//
// Usage:
//
//	s := rpc.NewRPCServer(
//		*config,
//		http.NewHttpServerTransport(),
//		serializer.NewJSONSerializer(),
//	)
//
//	if err := s.Serve(); err != nil {
//		panic(err)
//	 }
func NewRPCServer(
	config common.ServerConfig,
	transport transport.IRPCServerTransport,
	serializer serializer.IRPCSerializer,
) rpcServer {
	// https://github.com/golang/go/issues/17393
	if runtime.GOOS == "darwin" {
		signal.Ignore(syscall.Signal(0xd))
	}

	// Create shards map
	shardMap := xsync.NewMapOf[uint64, serverShard]()

	Logger.Infof("Created RPC Server")
	Logger.Infof(config.String())

	// Create the RPC server
	return rpcServer{
		config:     config,
		transport:  transport,
		serializer: serializer,
		shards:     shardMap,
	}
}

type rpcServer struct {
	config     common.ServerConfig
	transport  transport.IRPCServerTransport
	serializer serializer.IRPCSerializer
	shards     *xsync.MapOf[uint64, serverShard]
}

func (s *rpcServer) registerTransportHandler() {
	s.transport.RegisterHandler(func(shardId uint64, req []byte) []byte {
		var msg common.Message
		var respMsg common.Message

		// Get appropriate shard
		shard, ok := s.shards.Load(shardId)

		// Case shard does not exist -> error
		if !ok {
			respMsg = common.Message{
				MsgType: common.MsgTError,
				Err:     "shard not found",
			}
		} else {
			// Decode the request
			err := s.serializer.Deserialize(req, &msg)

			if err != nil {
				respMsg = common.Message{
					MsgType: common.MsgTError,
					Err:     fmt.Sprintf("failed to deserialize request: %s", err),
				}
			} else {
				// Let the adapter handle the request
				respMsg = *shard.Adapter.Handle(&msg, shard.Store)
			}
		}

		// Return result
		val, err := s.serializer.Serialize(respMsg)
		if err != nil {
			respMsg = common.Message{
				MsgType: common.MsgTError,
				Err:     fmt.Sprintf("failed to serialize response: %s", err),
			}
		}
		return val
	})
}

func (s *rpcServer) init() error {

	// Init logger
	common.InitLoggers(s.config)

	// Function to create a new database instance
	dbFactory := func() db.KVDB { return maple.NewMapleDB(nil) }

	// Create the Dragonboat NodeHost
	var nodeHost *dragonboat.NodeHost
	var err error
	if s.config.HasRemoteShard() {
		// Only create the NodeHost if we have remote shards
		nodeHost, err = dragonboat.NewNodeHost(s.config.ToNodeHostConfig())
		if err != nil {
			return fmt.Errorf("failed to create node host: %w", err)
		}
	}

	// Configure the timeout for the distributed store
	timeout := time.Duration(s.config.TimeoutSecond) * time.Second

	// CREATE SHARDS

	/*
		Note: A single RPC Server can have any number of remote and or local shards.
		Each shard can be a store or a lock manager. The following loop creates all
		the shards and stores them for the RPC server.
	*/

	for _, shardConfig := range s.config.Shards {

		// Case local store
		if shardConfig.Type == common.ShardTypeLocalIStore {
			s.shards.Store(shardConfig.ShardID, serverShard{
				Store:   lstore.NewLocalStore(dbFactory),
				Adapter: NewIStoreServerAdapter(),
			})
			Logger.Infof("created local store for shard %d", shardConfig.ShardID)

			// Case local lock
		} else if shardConfig.Type == common.ShardTypeLocalILockManager {
			s.shards.Store(shardConfig.ShardID, serverShard{
				Store:   lstore.NewLocalStore(dbFactory),
				Adapter: NewLockManagerServerAdapter(),
			})
			Logger.Infof("created local lock manager for shard %d", shardConfig.ShardID)

			// Case remote store or remote lock
		} else {
			if nodeHost == nil {
				return fmt.Errorf("node host is nil, cannot create remote store")
			}

			// Start Raft for the shard
			if err := nodeHost.StartConcurrentReplica(s.config.ClusterMembers, false, dstore.CreateStateMaschineFactory(dbFactory), s.config.ToDragonboatConfig(shardConfig.ShardID)); err != nil {
				Logger.Errorf("failed to start shard %v: %v", shardConfig.ShardID, err)
			}

			// Choose the appropriate adapter based on the shard type
			var adapter IRPCServerAdapter
			if shardConfig.Type == common.ShardTypeRemoteILockManager { // Case remote lock manager
				adapter = NewLockManagerServerAdapter()
			} else if shardConfig.Type == common.ShardTypeRemoteIStore { // Case remote store
				adapter = NewIStoreServerAdapter()
			} else {
				return fmt.Errorf("invalid shard type: %s", shardConfig.Type)
			}

			s.shards.Store(shardConfig.ShardID, serverShard{
				Store:   dstore.NewDistributedStore(nodeHost, shardConfig.ShardID, timeout),
				Adapter: adapter,
			})
		}
	}

	Logger.Infof("dKV setup completed successfully")

	// Configure the transport layer
	s.registerTransportHandler()

	return nil
}

// Serve starts the RPC server
// This function will also initialize the server plus the shards and start the transport layer
func (s *rpcServer) Serve() error {
	err := s.init()
	if err != nil {
		return err
	}
	return s.transport.Listen(s.config)
}

// temp
func runTests(nh *dragonboat.NodeHost) {

	// Start the pprof server
	go func() {
		Logger.Infof("Starting pprof server on :6060")
		Logger.Infof("%v", http.ListenAndServe(":6060", nil))
	}()

	Logger.Infof("sleeping before tests")
	time.Sleep(3 * time.Second)
	Logger.Infof("starting tests")

	shardID := uint64(128)

	for {

		readResult := testing.Benchmark(func(b *testing.B) {
			b.SetParallelism(10)
			// run test
			b.RunParallel(func(pb *testing.PB) {
				counter := 0
				for pb.Next() {

					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					_, err := nh.SyncRead(ctx, shardID, []byte{})
					cancel()
					if err != nil {
						fmt.Fprintf(os.Stdout, "failed to read, %v\n", err)
					}

					counter++
				}
			})
		})

		printResult("read", readResult)
	}
}

func printResult(test string, result testing.BenchmarkResult) {
	nsPerOp := float64(result.T.Nanoseconds()) / float64(result.N)
	opsPerSec := 1.0 / (nsPerOp / 1e9)

	// Format the time per operation with appropriate units
	var timePerOpStr string
	if nsPerOp < 1000 {
		timePerOpStr = fmt.Sprintf("%.2f ns/op", nsPerOp)
	} else if nsPerOp < 1000000 {
		timePerOpStr = fmt.Sprintf("%.2f ns/op (%.2f µs/op)", nsPerOp, nsPerOp/1000)
	} else if nsPerOp < 1000000000 {
		timePerOpStr = fmt.Sprintf("%.2f ns/op (%.2f ms/op)", nsPerOp, nsPerOp/1000000)
	} else {
		timePerOpStr = fmt.Sprintf("%.2f ns/op (%.2f s/op)", nsPerOp, nsPerOp/1000000000)
	}

	// Format the operations per second with appropriate units
	opsPerSecStr := fmt.Sprintf("%.2f ops/sec", opsPerSec)

	// Print the formatted result
	fmt.Printf("%-20s\t%s\t%s\tAllocs: %d, AllocBytes: %d", test, timePerOpStr, opsPerSecStr, result.AllocsPerOp(), result.AllocedBytesPerOp())

	fmt.Println()
}
