// Package lstore implements a local, in-memory, single-node key-value store based on the
// store.IStore interface. It provides a thin wrapper around any db.KVDB
// implementation with automatic write index management. Data is stored entirely
// in memory and is not persisted between process restarts.
//
// Key Features:
//   - Pure in-memory storage without persistence
//   - Direct integration with db.KVDB implementations
//   - Automatic write index progression using atomic operations
//   - Feature detection to handle unsupported operations gracefully
//   - Thread-safe operations for concurrent access
//
// Implementation Details:
//
//   - Write Index Management: The store maintains an atomic counter that automatically
//     increments with each write operation. This provides a monotonically increasing
//     logical timestamp that ensures consistent ordering of operations and enables
//     time-based features like expiration and deletion.
//
//   - Feature Detection: Before executing operations, the store checks if the underlying
//     db.KVDB implementation supports the requested feature through the SupportsFeature
//     method. Unsupported operations return appropriate error codes rather than failing
//     silently or producing undefined behavior.
//
//   - Composition Architecture: The store follows a composition pattern where the
//     store.DBFactory factory function injects the underlying db.KVDB implementation.
//     This allows the store to work with any db.KVDB-compatible engine without modification.
//
// Thread Safety:
//
//	All operations in the local store are thread-safe. The write index is managed
//	using atomic operations that guarantee correct behavior even under concurrent access.
//	The underlying db.KVDB implementation is expected to provide its own thread safety
//	guarantees for the actual storage operations.
//
// Usage Example:
//
//	// Create a store with a maple database backend
//	factory := func() db.KVDB { return maple.NewDB(maple.DefaultOptions()) }
//	store := lstore.NewLocalStore(factory)
//
//	// Store a value with 5-minute expiration
//	err := store.SetE("session:123", sessionData, 300, 0)
//
//	// Retrieve the value
//	value, exists, err := store.Get("session:123")
//
// Suitable Use Cases:
//
//	The local store is ideal for:
//	- Ephemeral data that doesn't need to survive process restarts
//	- Single-node applications where distributed consensus is not required
//	- Testing and development environments
//	- Runtime caching and session storage within a single process
//
// Performance Considerations:
//
//	The local store adds minimal overhead to the underlying db.KVDB implementation.
//	The primary additional cost is the atomic increment operation for write index
//	management, which typically has negligible impact on performance compared to
//	the actual storage operations.
//
// For distributed scenarios requiring consensus across multiple nodes, consider
// using the dstore package instead, which provides a RAFT-based implementation
// of the same interface with strong consistency guarantees.
package lstore
