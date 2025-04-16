package serve

import (
	"fmt"
	cmdUtil "github.com/ValentinKolb/dKV/cmd/util"
	"github.com/ValentinKolb/dKV/lib/db/util"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/serializer"
	"github.com/ValentinKolb/dKV/rpc/server"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/ValentinKolb/dKV/rpc/transport/http"
	"github.com/ValentinKolb/dKV/rpc/transport/tcp"
	"github.com/ValentinKolb/dKV/rpc/transport/unix"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strconv"
	"strings"
)

var (
	serveCmdConfig = &common.ServerConfig{}
	ServeCmd       = &cobra.Command{
		Use:     "serve",
		Short:   "Start the dKV server",
		Long:    `Start the dKV server with the specified configuration. The configuration can be set via command line flags or environment variables. The format of the environment variables is DKV_<flag> (e.g. DKV_TIMEOUT_SECOND=15)`,
		PreRunE: processConfig,
		RunE:    run,
	}
)

func init() {
	// initialize viper
	cobra.OnInitialize(initConfig)

	// add flags
	key := "shards"
	ServeCmd.PersistentFlags().String(key, "100=lstore,200=lockmgr(lstore)", cmdUtil.WrapString("Comma-separated list of shards to serve. Format: ID=TYPE where TYPE is one of: dstore, lstore, lockmgr(dstore), lockmgr(lstore)"))

	key = "rtt-millisecond"
	ServeCmd.PersistentFlags().Int(key, 100, cmdUtil.WrapString("(ConfServerModeMultiNode Mode) RTTMillisecond defines the average Round Trip Time (RTT) in milliseconds between two NodeHost instances. \nOther raft configuration parameters (ElectionRTT=value/10, HeartbeatRTT=value/100) are derived from this value"))

	key = "snapshot-entries"
	ServeCmd.PersistentFlags().Int(key, 10, cmdUtil.WrapString("(ConfServerModeMultiNode Mode) SnapshotEntries defines how often the state machine should be snapshotted automatically. It is defined in terms of the number of applied Raft log entries. SnapshotEntries can be set to 0 to disable such automatic snapshotting (not recommended)"))

	key = "compaction-overhead"
	ServeCmd.PersistentFlags().Int(key, 5, cmdUtil.WrapString("(ConfServerModeMultiNode Mode) CompactionOverhead defines the number of snapshots that should be retained in the system. When a new snapshot is generated, the system will attempt to remove older snapshots that go beyond the specified number of retained snapshots. Recommended value is about 1/2 of SnapshotEntries"))

	key = "data-dir"
	ServeCmd.PersistentFlags().String(key, "data", cmdUtil.WrapString("(ConfServerModeMultiNode Mode) DataDir is the directory used for storing the snapshots"))

	key = "replica-id"
	ServeCmd.PersistentFlags().String(key, "", cmdUtil.WrapString("(ConfServerModeMultiNode Mode) ReplicaID is the unique identifier for this NodeHost instance (e.g. 'node-1')"))

	key = "cluster-members"
	ServeCmd.PersistentFlags().String(key, "", cmdUtil.WrapString("(ConfServerModeMultiNode Mode) ClusterMembers is a comma-separated list of NodeHost addresses in the format 'node-1=localhost:63001,node-2=localhost:63002,...'"))

	key = "timeout"
	ServeCmd.PersistentFlags().Int64(key, 5, cmdUtil.WrapString("(ConfServerModeMultiNode Mode) Timeout in seconds"))

	key = "endpoint"
	ServeCmd.PersistentFlags().String(key, "0.0.0.0:8080", cmdUtil.WrapString("The address on which the API will listen (e.g. http:localhost:8080, /tmp/dkv.sock, ...)"))

	key = "log-level"
	ServeCmd.PersistentFlags().String(key, "info", cmdUtil.WrapString("LogLevel is the level at which logs will be output (debug, info, warn, error)"))
}

// processConfig reads the configuration from the command line flags and environment variables and converts them to the server configuration
func processConfig(cmd *cobra.Command, _ []string) error {
	// bind the flags to viper
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	// parse shards
	shardsConfig := viper.GetString("shards")
	serveCmdConfig.Shards = []common.ServerShard{}
	for _, shardConfig := range strings.Split(shardsConfig, ",") {
		parts := strings.Split(shardConfig, "=")
		if len(parts) != 2 {
			return fmt.Errorf("invalid shard format: %s (expected ID=TYPE)", shardConfig)
		}

		// Parse shard ID
		shardID, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid shard ID %s: %v", parts[0], err)
		}

		// Parse shard type
		shardType := strings.TrimSpace(parts[1])
		var serverShardType common.ServerShardType

		switch shardType {
		case "dstore":
			serverShardType = common.ShardTypeRemoteIStore
		case "lstore":
			serverShardType = common.ShardTypeLocalIStore
		case "lockmgr(dstore)":
			serverShardType = common.ShardTypeRemoteILockManager
		case "lockmgr(lstore)":
			serverShardType = common.ShardTypeLocalILockManager
		default:
			return fmt.Errorf("invalid shard type: %s (expected one of: dstore, lstore, lockmgr(dstore), lockmgr(lstore))", shardType)
		}

		serveCmdConfig.Shards = append(serveCmdConfig.Shards, common.ServerShard{
			ShardID: shardID,
			Type:    serverShardType,
		})
	}

	// read the configuration from the command line flags and environment variables
	serveCmdConfig.RTTMillisecond = viper.GetUint64("rtt-millisecond")
	serveCmdConfig.SnapshotEntries = viper.GetUint64("snapshot-entries")
	serveCmdConfig.CompactionOverhead = viper.GetUint64("compaction-overhead")
	serveCmdConfig.DataDir = viper.GetString("data-dir")
	serveCmdConfig.TimeoutSecond = viper.GetInt64("timeout")
	serveCmdConfig.Endpoint = viper.GetString("endpoint")
	serveCmdConfig.LogLevel = viper.GetString("log-level")

	// parse replica id
	if id := viper.GetString("replica-id"); id != "" {
		serveCmdConfig.ReplicaID = uint64(util.HashString(id, 0))
	} else if serveCmdConfig.HasRemoteShard() {
		// error only if cluster mode
		return fmt.Errorf("ReplicaId is required for remote shards")
	}

	// parse cluster members
	if clusterMembers := viper.GetString("cluster-members"); clusterMembers != "" {
		serveCmdConfig.ClusterMembers = make(map[uint64]string)
		for _, member := range strings.Split(clusterMembers, ",") {
			parts := strings.Split(member, "=")
			if len(parts) != 2 {
				return fmt.Errorf("invalid cluster member format: %s (expected ID=address)", member)
			}
			idHash := util.HashString(parts[0], 0)
			serveCmdConfig.ClusterMembers[uint64(idHash)] = parts[1]
		}
	} else if serveCmdConfig.HasRemoteShard() {
		// error only if cluster mode
		return fmt.Errorf("ClusterMembers is required for remote shards")
	}

	// test if the replica id is in the cluster members (only for cluster mode)
	if _, ok := serveCmdConfig.ClusterMembers[serveCmdConfig.ReplicaID]; !ok && serveCmdConfig.HasRemoteShard() {
		return fmt.Errorf("no address found for replica ID %d in cluster members", serveCmdConfig.ReplicaID)
	}

	return nil
}

// serve starts the dKV server
func run(_ *cobra.Command, _ []string) error {

	// parse the serializer
	var s serializer.IRPCSerializer
	switch viper.GetString("serializer") {
	case "json":
		s = serializer.NewJSONSerializer()
	case "gob":
		s = serializer.NewGOBSerializer()
	case "binary":
		s = serializer.NewBinarySerializer()
	default:
		return fmt.Errorf("invalid serializer %s", viper.GetString("serializer"))
	}

	// Parse the transport
	var t transport.IRPCServerTransport
	switch viper.GetString("transport") {
	case "http":
		t = http.NewHttpServerTransport()
	case "tcp":
		t = tcp.NewTCPServerTransport(64 * 1024)
	case "unix":
		t = unix.NewUnixServerTransport(64 * 1024)
	default:
		return fmt.Errorf("invalid transport %s", viper.GetString("transport"))
	}

	serv := server.NewRPCServer(
		*serveCmdConfig,
		t,
		s,
	)

	return serv.Serve()
}

// initConfig reads in serveCmdConfig file and ENV variables if set.
func initConfig() {
	// load env files
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")

	// initialize viper
	viper.SetEnvPrefix("dkv")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match

}
