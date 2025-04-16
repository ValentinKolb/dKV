// Package dstore implements a distributed, fault-tolerant key-value store using
// the Dragonboat RAFT consensus library. It provides a strongly consistent implementation
// of the store.IStore interface that can operate across multiple nodes while
// maintaining linearizable consistency.
//
// Architecture:
//
// The dstore implementation consists of three main components:
//
//   - Store Client: Implements the store.IStore interface and communicates with
//     the RAFT cluster. It serializes operations into commands, sends them to the
//     consensus layer, and processes responses.
//
//   - State Machine: A Dragonboat IConcurrentStateMachine implementation that processes
//     commands and queries on each node. The state machine contains the actual db.KVDB
//     instance and applies operations to it.
//
//   - Communication Protocol: Defined in the internal package, this consists of Command
//     and Query structures with serialization logic for transmitting operations across
//     the network.
//
// Consensus Model:
//
//	The store uses Dragonboat's implementation of the RAFT consensus protocol which provides:
//
//	- Strong Consistency: All operations are linearizable, meaning they appear to
//	  execute atomically and in a consistent order across all nodes.
//
//	- Fault Tolerance: The system remains operational as long as a majority of nodes
//	  are functioning. With 2N+1 nodes, up to N node failures can be tolerated.
//
//	- Leader-Based Processing: Write operations are forwarded to the leader node,
//	  replicated to followers, and only considered committed when a majority of nodes
//	  have persisted the operation.
//
// Write Operations:
//
//	All write operations (Set, SetE, SetEIfUnset, Expire, Delete) follow this flow:
//
//	1. The operation is serialized into a Command structure
//	2. The Command is proposed to the RAFT cluster via SyncPropose
//	3. The leader node replicates the command to a majority of followers
//	4. Once committed, the command is executed on the state machine on each node (Update method in statemachine.go)
//	5. The result (ACK) is returned to the client
//
//	The write index for all operations is provided by the RAFT log index, ensuring
//	a globally consistent ordering of operations across the cluster.
//
// Read Operations:
//
// Read operations (Get, Has, GetDBInfo) can be handled in two ways:
//
//   - Linearizable Reads: By default, reads use SyncRead which ensures that the node
//     processing the read has applied all committed log entries locally before processing
//     the request. This guarantees the operation sees the latest committed state of the
//     database, regardless of which node in the cluster processes the read.
//
//   - Stale Reads: For less critical operations (GetDBInfo), StaleRead is used,
//     which may return slightly outdated information but with lower latency.
//
// Error Handling and Retries:
//
//	The store implements automatic retry logic for transient failures:
//
//	- System Busy: When Dragonboat returns ErrSystemBusy, the operation is retried
//	  after a short delay, up to a configurable number of attempts.
//
//	- Timeouts: All operations have a configurable timeout. If consensus cannot be
//	  reached within this period, the operation fails with a timeout error.
//
//	- Feature Compatibility: Before executing operations, the state machine verifies
//	  that the underlying db.KVDB implementation supports the required features.
//
// Snapshotting and Recovery:
//
// The state machine implements Dragonboat's snapshotting interface to persist its state:
//
//   - Fuzzy Snapshots: The state machine creates snapshots without pausing operations,
//     leveraging the db.KVDB's Save method.
//
//   - Recovery: On startup or when joining a cluster, nodes first restore their state
//     from the most recent (fuzzy )snapshot using the db.KVDB's Load method. Then, they receive
//     all RAFT log entries that were committed after the snapshot was created from other
//     nodes in the cluster. This two-phase process ensures that after recovery is complete,
//     the node reaches the same consistent state as all other nodes in the cluster.
//
// Usage:
//
//	Setting up and using dstore requires several steps:
//
//	1. Initialize Dragonboat NodeHost (RAFT client)
//	2. Create a db.KVDB factory function
//	3. Start a RAFT replica with the state machine factory
//	4. Create the distributed store with appropriate timeout
//	5. Begin operations once the shard is ready
//
//	Example:
//
//	  // Create NodeHost (RAFT client)
//	  nh, err := dragonboat.NewNodeHost(nodeHostConfig)
//	  if err != nil { ... }
//
//	  // DB factory for store
//	  dbFactory := func() db.KVDB { return maple.NewMapleDB(nil) }
//
//	  // Create and start shard (RAFT server)
//	  err := nh.StartConcurrentReplica(
//	      clusterMembers,
//	      false,
//	      dstore.CreateStateMaschineFactory(dbFactory),
//	      shardConfig)
//	  if err != nil { ... }
//
//	  // Create store with appropriate timeout
//	  timeout := time.Duration(5) * time.Second
//	  store := dstore.NewDistributedStore(nh, shardID, timeout)
//
//	  // Wait for shard readiness then begin operations
//	  // ...
//
// Performance Considerations:
//
//   - Consensus Overhead: Due to the requirement for replication and majority commitment,
//     distributed operations are significantly slower than local operations.
//
//   - Network Conditions: Operation latency is highly dependent on network conditions
//     between nodes. Timeouts should be adjusted based on expected network performance.
//
// Deployment Recommendations:
//
//   - Node Count: Deploy with an odd number of nodes (typically 3, 5, or 7) to ensure
//     majority consensus is always possible.
//
//   - Geographic Distribution: For maximum fault tolerance, distribute nodes across
//     different failure domains (servers, racks, data centers).
//
//   - Network Quality: Ensure low-latency, high-bandwidth connections between nodes
//     for optimal performance.
//
// Limitations:
//
//   - Majority Requirement: Operations cannot proceed if a majority of nodes are unavailable
//   - Leader Dependency: Write operations require the leader to be available
//   - Consistency vs. Performance: The strong consistency model introduces performance overhead
//
// For scenarios where distributed consensus is not required, consider using the simpler
// and faster lstore package, which provides a single-node not-persistent implementation of the
// same interface.
package dstore
