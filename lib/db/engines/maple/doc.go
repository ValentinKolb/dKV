// Package maple implements a high-performance key-value database (KVDB) with
// advanced concurrency control and time-based operations. It provides a complete
// implementation of the db.KVDB interface with a focus on thread safety,
// performance, and memory efficiency.
//
// The package focuses on:
//   - Optimized concurrent access through sharding and lockmgr-free data structures
//   - Time-based entry management with expiration and deletion capabilities
//   - Efficient garbage collection to reclaim memory from expired/deleted entries
//     without impacting performance
//   - Persistent storage with fuzzy snapshots and efficient binary encoding
//   - Comprehensive metrics and statistics for monitoring and optimization
//
// Key Components:
//
//   - mapleImpl: The central database structure implementing db.KVDB. It manages shards,
//     coordinates garbage collection, and provides the public API for key-value
//     operations. The mapleImpl uses atomic-like operations for thread safety and maintains a
//     monotonically increasing write index for consistent operation ordering.
//     The mapleImpl structure does not manage the write index itself, but rather delegates
//     this responsibility to the caller to allow for flexible integration with
//     external systems. That way, the caller can choose the best way to generate
//     write indices based on their requirements (e.g. logical timestamps, Lamport, ...)
//
//   - Shard: A partition of the database that manages a subset of the key space.
//     Each shard contains its own data map and priority queues for expiration and
//     deletion. Shards operate independently to minimize lockmgr contention and
//     enable high concurrency. Keys are dstore across shards using a hash
//     function to ensure even distribution.
//
//   - Entry: The core structure for storing values and metadata. Each entry
//     contains the byte value, expiration timestamp, deletion timestamp, and
//     creation index. This structure enables efficient time-based operations
//     and supports the stale write detection mechanism.
//
//   - Event System: A lockmgr-free multi-producer single-consumer event queue that
//     coordinates garbage collection operations across shards. Events are
//     generated when entries are set, expired, or deleted, and processed by the
//     garbage collector to update the priority queues.
//
// Internal Mechanisms:
//
//   - Sharding Strategy: Keys are dstore across shards in a two-step process:
//     1. String keys are converted to 64-bit integers using the HashString function
//     with a database-specific seed
//     2. The integer key is right-shifted by 7 bits to use higher-quality bits for
//     distribution
//     This strategy ensures even distribution with minimal hash computation overhead.
//
//   - Write Index: A logical timestamp that orders operations in the database.
//     The index is monotonically increased and atomically updated using
//     CompareAndSwap operations. It serves multiple purposes:
//     1. Detecting and rejecting stale writes (operations with lower indices)
//     2. Determining when entries should expire or be deleted
//     3. Tracking the logical time of the database for operations
//
//   - Time-based Operations: When an entry is set, the caller can specify
//     expiration and deletion times. The specified times are relative to the
//     current write index, allowing for flexible time-based operations.
//     The database supports two distinct time-based mechanisms:
//     1. Expiration (expireIn): Marks the entry as expired but keeps it in
//     the database. Expired entries return false for Get() but true for Has().
//     2. Deletion (deleteIn): Schedules the entry for complete removal from
//     the database. Deleted entries return false for both Get() and Has().
//     These mechanisms use the write index as a logical clock, with operations
//     taking effect when the current index reaches the specified threshold.
//
//   - Stale Write Prevention: Each entry stores the write index at which it was
//     created or last updated. A write operation is only applied if its write
//     index is greater than or equal to the stored index of the entry. This ensures
//     that out-of-order or delayed operations do not overwrite newer data and
//     provides a form of causal consistency.
//
//   - Conditional Writes:
//     The SetEIfUnset operation allows for atomic compare-and-set
//     operations that only modify a key if it doesn't exist. This provides
//     primitive support for dstore coordination patterns.
//
//   - Lock-free Data Structures:
//     The implementation uses specialized lockmgr-free
//     or lockmgr-minimizing data structures:
//     1. xsync.MapOf: A concurrent map implementation that itself shards keys internally
//     for efficient access and minimal locking
//     2. util.MapHeap: A priority queue for tracking expiration and deletion times
//     3. util.LockFreeMPSC: A lockmgr-free queue for efficient event communication
//     These structures minimize contention and enable high throughput under
//     concurrent load.
//
//   - Persistence Format: The database uses a compact binary format with the
//     following structure:
//     1. Magic number "MAPLEDB\x00" to identify the file format
//     2. Version number (currently 3)
//     3. Database seed value for hash function consistency
//     4. Number of entries
//     5. For each entry: key, expiration time, deletion time, index, value length,
//     value bytes
//     Note: The database does not lockmgr the database during snapshot creation or loading.
//     Instead, it creates a fuzzy snapshot that does not represent a consistent cut of
//     the database. It also assumes that the caller provides a consistent snapshot for
//     loading. It is on the caller to ensure that the snapshot is consistent.
//
//   - Metrics and Monitoring: The database provides detailed statistics via the
//     GetInfo method, including:
//     1. Size estimates based on sampling and statistical analysis
//     2. Shard distribution statistics to detect imbalances
//     3. GC backlog information to monitor collection efficiency
//     4. Feature capability reporting via the SupportsFeature method
//
// Garbage Collection:
//
//   - To minimize the impact on performance, a special garbage collection system
//     is used that operates without holt-the-world pauses. The garbage collection
//     is run in a single goroutine per shard.
//
//   - The GC system works as follows:
//     1. Set: The entry is created or updated with a new value and a write index. If
//     the Entry expires or should be deleted,
//     an event is created and added to the event queue of the shard.
//     2. The GC goroutine processes the event queue, updating the expiration and deletion
//     heaps. These heaps are only ever accessed by the shard-specific GC goroutine and never
//     concurrently. That way, the GC goroutine can safely update the heaps without any
//     locks (speeding up the process).
//     3. After updating the heaps, the GC goroutine looks for entries that have reached their
//     expiration time and frees up the storage by setting the value to nil. Thereafter, it looks
//     for entries that have reached their deletion time and removes them from the database.
//     After that the GC goroutine waits for the next events.
//
// The Delete() method also creates an event on the shard's event queue, which
// leads to the GC goroutine removing the entry from the GC heaps (and thereby not tracking
// it anymore).
//
// Since the Database can not guarantee that the GC removes the entry immediately,
// the Get() and Has() methods check if the entry is expired or deleted and return
// false if it is. This way, even if the internal state of the database is not up-to-date,
// the user can still rely on the time-based operations.
//
// The GC system carefully handles edge cases such as concurrent updates during
// collection.
//
// This implementation offers several advantages:
//   - High throughput for concurrent operations on different keys
//   - Efficient memory usage through automatic garbage collection
//   - Flexible time-based operations for expiration and deletion policies
//   - Thread-safe operations without excessive locking
//   - Persistent storage for recovery and data migration
//   - Detailed metrics for monitoring and performance tuning
//
// The maple package is designed to serve as a backend for applications requiring
// a high-performance key-value store with time-based operations, such as caches,
// session stores, and temporary data storage systems.
package maple
