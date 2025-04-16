// Package db provides a standardized interface for key-value database implementations.
// It defines a comprehensive KVDB interface that allows for consistent interaction
// with various database backends while abstracting implementation details.
//
// The package focuses on:
//   - A unified interface for key-value operations
//   - Feature discovery through capability flags
//   - Standardized persistence operations
//   - Comprehensive metadata reporting
//
// Key Components:
//
//   - KVDB Interface: The core interface that all database implementations must satisfy.
//     It provides methods for basic operations (Set, Get, Has, Delete),
//     time-based operations (SetE, Expire, GarbageCollect),
//     specialized operations (SetEIfUnset), metadata retrieval (GetDBInfo),
//     and persistence operations (Save, Load).
//
//   - Feature Flags: The Feature type defines capability flags that implementations
//     can advertise through the SupportsFeature method. This allows clients to
//     discover supported operations at runtime.
//
//   - Implementation Identifiers: The Implementation type provides string constants
//     for different database backends (currently "maple").
//
//   - Database Information: The DatabaseInfo structure provides standardized
//     reporting on database state, including size statistics, implementation type,
//     and implementation-specific metadata. Note: For most implementations all
//     size statistics will be estimated since a precise calculation can be
//     expensive.
//
// This interface-driven approach allows applications to:
//   - Swap database implementations without code changes
//   - Gracefully handle operations not supported by specific implementations
//   - Maintain consistent behavior across different storage backends
//   - Collect standardized metrics for monitoring and management
//
// Note on Time-Based Operations:
//   - Write Operations and Time-Tracking: All write operations require a write-index parameter
//     that serves as a logical timestamp. This write-index is used to:
//     1. Record when an entry was created or modified
//     2. Calculate expiration and deletion times (by adding offsets to the current write-index)
//     3. Update the database's global logical clock
//   - Read Operations: Read methods do not accept a time-index parameter as they always operate
//     against the most recently set write-index. This design assumes reads occur after
//     the global time has been properly advanced through writes.
//   - Manual Time Advancement: If the caller needs to advance the logical time without performing
//     a write operation, the SetWriteIdx() method should be used.
//   - Monotonicity Guarantee: All implementations must ensure that the write-index only increases
//     monotonically. Attempts to set a write-index lower than the current one must be ignored
//     to maintain temporal consistency.
//
// Note on Garbage Collection:
//   - All implementations must support garbage collection and ensure that deleted entries
//     are eventually removed from the database to prevent memory leaks.
//   - External Consistency: Implementations must maintain strong external consistency
//     regardless of their internal garbage collection state:
//   - Get() must never return an entry that has logically expired, even if the entry
//     still exists internally pending collection.
//   - Has() must never return true for an entry that has been logically deleted, even if
//     the entry still exists internally pending collection.
//   - This separation between logical state (expired/deleted) and physical state (still present
//     in memory) allows implementations to use efficient background collection strategies
//     without compromising the consistency guarantees of the interface.
//
// Related Packages:
//
// The engines/maple package (github.com/ValentinKolb/dKV/lib/db/engines/maple) provides a
// high-performance implementation of the KVDB interface using a sharded in-memory architecture.
// It features advanced concurrency support through lockmgr-free data structures, comprehensive
// time-based operations for key expiration and deletion, efficient background garbage collection,
// and binary persistence capabilities. The implementation is optimized for scenarios requiring
// high throughput with concurrent operations while maintaining strong consistency guarantees.
//
// The util package (github.com/ValentinKolb/dKV/lib/util) provides complementary
// tools for working with db.KVDB implementations:
//   - SizeHistogram: Utilities for analyzing data size distributions
//   - MapHeap: A priority queue implementation for memory management and garbage collection
//   - LockFreeMPSC: A lockmgr-free multi-producer single-consumer queue for concurrent operations
//   - ... and more
//
// The testing package (github.com/ValentinKolb/dKV/lib/db/testing) provides
// standardized tests and benchmarks for database implementations that satisfy the db.KVDB interface.
//   - RunKVDBTests: Runs a standardized test suite to validate implementations
//   - RunKVDBBenchmarks: Provides performance benchmarks for comparing implementations
package db
