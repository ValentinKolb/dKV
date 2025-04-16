// Package server implements the RPC server for the distributed key-value store system.
// It provides adapters for handling RPC requests to both store and lock manager services,
// along with the core server implementation that manages shards and request routing.
//
// The package focuses on:
//   - Server-side RPC request handling for both store and lock manager operations
//   - Adapter pattern to decouple application logic from RPC mechanisms
//   - Flexible shard configuration with support for local and distributed stores
//   - Dynamic creation of stores and lock managers based on shard configuration
//
// Key Components:
//
//   - IRPCServerAdapter: Interface defining the contract for all server adapters,
//     with the Handle method that processes incoming requests against a store.IStore.
//
//   - NewIStoreServerAdapter: Factory function creating an adapter for key-value
//     store operations, translating RPC requests to store.IStore method calls.
//
//   - NewLockManagerServerAdapter: Factory function creating an adapter for distributed
//     locking operations, creating a lockmgr.ILockManager on top of the store.
//
//   - NewRPCServer: Factory function creating a configured server with the specified
//     transport and serializer mechanisms.
//
// Usage Example:
//
//	// Create server configuration
//	config := common.ServerConfig{
//	  Shards: []common.ServerShard{
//	    {ShardID: 100, Type: common.ShardTypeLocalIStore},
//	    {ShardID: 200, Type: common.ShardTypeLocalILockManager},
//	  },
//	  Endpoint: "0.0.0.0:8080",
//	  TimeoutSecond: 5,
//	  LogLevel: "info",
//	}
//
//	// Create and start the server
//	s := server.NewRPCServer(
//	  config,
//	  tcp.NewTCPDefaultServerTransport(),
//	  serializer.NewBinarySerializer(),
//	)
//
//	// Start the server
//	if err := s.Serve(); err != nil {
//	  log.Fatalf("Server error: %v", err)
//	}
//
// The server supports four types of shards, which can be mixed within a single server:
//
//   - ShardTypeLocalIStore: A local store implementation, suitable for single-node deployments
//     or development environments.
//
//   - ShardTypeRemoteIStore: A distributed store implementation using Raft consensus,
//     providing strong consistency across multiple nodes. When using this type,
//     RAFT configuration (RTTMillisecond, SnapshotEntries, CompactionOverhead,
//     DataDir, ReplicaID, and ClusterMembers) must be properly configured.
//
//   - ShardTypeLocalILockManager: A local lock manager implementation, using a local
//     store as its backend.
//
//   - ShardTypeRemoteILockManager: A distributed lock manager implementation using
//     a distributed store as its backend. When using this type, all RAFT configuration
//     parameters must be properly configured.
//
// Thread Safety:
//
//	The server implementation is thread-safe and can handle concurrent requests
//	Across multiple connections. Each request is processed independently.
//	The Listen method is not thread-safe and should be called only once.
package server
