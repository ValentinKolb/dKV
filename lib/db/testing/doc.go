// Package testing provides standardised tests and benchmarks for
// database implementations that satisfy the db.KVDB interface.
//
// The package contains:
//   - testing: A comprehensive test suite for validating conformance to the KVDB interface contract
//   - benchmark: Performance tests for measuring throughput of common database operations
//
// This package is particularly useful for:
//   - Applications that need to select the most appropriate database implementation
//     based on performance characteristics
//   - Database developers implementing the KVDB interface
//
// Example usage:
//
//	// Creating a factory function for your implementation
//	factory := func() db.KVDB {
//		return NewMyDatabase()
//	}
//
//	// Running the standard test suite
//	util.RunKVDBTests(t, "MyDatabase", factory)
//
//	// Running performance benchmarks
//	util.RunKVDBBenchmarks(b, "MyDatabase", factory)
package testing
