// Package lockmgr implements a locking mechanism using
// key-value stores that implement the store.IStore interface. It provides
// a simple yet robust way to coordinate access to shared resources across
// multiple processes or nodes.
//
// The lockmgr only ever stores in the provides IStore and has no other internal
// state. Therefor it is safe to be created multiple times on the same store.
// It is event possible to create a new lockmgr for every acquire and or release
// operation. As long as the same store is used every time, all locks will
// work as expected.
//
// Core Functionality:
//   - Lock acquisition with ownership verification
//   - Automatic lockmgr expiration through configurable timeouts
//   - Safe release operations that verify ownership
//
// Implementation Approach:
//
//	Locks are implemented by leveraging the atomic conditional operations
//	of the underlying store. Specifically:
//
//	- Lock Acquisition: Attempts to create a key using SetEIfUnset, which
//	  guarantees that only one requester can successfully create the key.
//	  The value contains a randomly generated owner ID that identifies the
//	  lockmgr holder.
//
//	- Lock Verification: A successful SetEIfUnset operation is followed by
//	  a Get operation to confirm the lockmgr was acquired by checking that the
//	  stored value matches the owner ID.
//
//	- Timeouts: Locks can be configured with an optional timeout (deleteIn)
//	  that automatically releases the lockmgr after the specified period,
//	  preventing deadlocks if a client crashes.
//
//	- Safe Release: The ReleaseLock operation first verifies that the
//	  requester is the legitimate owner of the lockmgr by comparing owner IDs
//	  before executing the Delete operation.
//
// Thread Safety:
//
//	The lockmgr is as thread-safe as the underlying store.IStore
//	implementation. All operations are performed through the store interface,
//	which typically provides thread safety guarantees.
//
// Distributed Considerations:
//
//	When used with a distributed store implementation like dstore, the
//	lockmgr provides true distributed locking with consensus-based
//	guarantees. This enables coordination across multiple nodes in a cluster
//	while maintaining strong consistency properties.
//
// Usage Example:
//
//	// Create a lockmgr provider with a store backend
//	lockProvider := lockmgr.NewLockManager(store)
//
//	// Acquire a lockmgr with a timeout
//	acquired, ownerID, err := lockProvider.AcquireLock("resource:123", 30)
//	if err != nil {
//	    // Handle error
//	}
//
//	if acquired {
//	    // Use the resource safely
//	    // ...
//
//	    // Release the lockmgr when done
//	    released, err := lockProvider.ReleaseLock("resource:123", ownerID)
//	    if err != nil {
//	        // Handle error
//	    }
//	}
//
// Security Considerations:
//
//	The lockmgr mechanism uses randomly generated owner IDs, which provides
//	reasonable protection against accidental lockmgr stealing. However, it is
//	not designed to resist malicious attacks, as an attacker with access to
//	the underlying store could potentially manipulate lockmgr data directly.
//
// Performance Impact:
//
//	Lock operations require 1-2 store operations each:
//	- AcquireLock: One SetEIfUnset followed by one Get
//	- ReleaseLock: One Get followed by a conditional Delete
//
//	The performance characteristics therefore depend primarily on the
//	underlying store implementation.
package lockmgr
