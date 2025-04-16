// Package client implements RPC clients for the distributed key-value store system.
// It provides implementations of the store.IStore and lockmgr.ILockManager interfaces
// that communicate with remote servers via RPC.
//
// The package focuses on:
//   - Transparent RPC access to store and lock manager implementations
//   - Integration with the transport and serialization layers
//   - Error handling and conversion between RPC and domain errors
//
// Key Components:
//
//   - NewRPCStore: Factory function that creates a client implementing the store.IStore
//     interface. This client forwards all operations to remote servers via the configured
//     transport layer.
//
//   - NewRPCLockMgr: Factory function that creates a client implementing the
//     lockmgr.ILockManager interface for distributed locking operations.
//
// Usage Example:
//
//		// Configure the client
//		util := common.ClientConfig{
//		  Endpoints:              []string{"localhost:5000"},
//		  TimeoutSecond:          5,
//		  RetryCount:             3,
//		  ConnectionsPerEndpoint: 1,
//		}
//
//	 // Create a serializer
//		serializer := serializer.NewBinarySerializer()
//
//		// Create store client
//		store, _ := client.NewRPCStore(1, util, tcp.NewTCPClientTransport(), serializer)
//
//		// Use the store
//		store.Set("mykey", []byte("myvalue"))
//		value, exists, _ := store.Get("mykey")
//
//		// Create and use a lock manager
//		lockMgr, _ := client.NewRPCLockMgr(2, util, tcp.NewTCPClientTransport(), serializer)
//		acquired, ownerID, _ := lockMgr.AcquireLock("mylock", 30)
//		if acquired {
//		  lockMgr.ReleaseLock("mylock", ownerID)
//		}
//
// Performance Considerations:
//
//   - For applications that frequently send large payloads, increasing ConnectionsPerEndpoint
//     can improve throughput by allowing parallel requests.
//
//   - For small messages, a single connection per endpoint is often more efficient due to
//     reduced connection overhead.
//
//   - The choice of serializer significantly affects performance. The binary serializer
//     provides the best performance and smallest payload size.
//
// Thread Safety:
//
//	All client implementations are thread-safe and can be used concurrently from
//	multiple goroutines without additional synchronization.
package client
