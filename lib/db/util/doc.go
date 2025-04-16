// Package util provides utility components for
// database implementations that satisfy the db.KVDB interface.
//
// The package contains:
//   - statistics: Utility tools for analyzing database characteristics and a SizeHistogram for tracking data size distribution
//   - functions: Hash functions and other utility functions
//   - mapheap: A priority queue implementation for garbage collection that also supports key-based access
//   - lockfreempsc: A lockmgr-free Multi-Producer Single-Consumer (MPSC) queue implementation build for high throughput and low latency
//
// This package is particularly useful for:
//   - Database developers implementing the KVDB interface
//   - Implementation of garbage collection or other priority queue systems
//   - Monitoring systems that need to track database size and distribution metrics
//
// Each component is designed to work with any implementation of the db.KVDB interface,
// allowing for consistent validation and measurement across different storage backends.
package util
