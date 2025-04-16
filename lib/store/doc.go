// Package store provides a high-level interface for key-value storage operations
// with advanced features like expiration, deletion scheduling, and unified error handling.
// It serves as an abstraction layer over the lower-level db.KVDB implementations, adding
// functionality such as write index management and standardized error reporting.
//
// The package focuses on:
//   - A unified interface (IStore) for key-value operations across different backends
//   - Pluggable storage backend architecture through DBFactory pattern
//
// Key Components:
//
//   - IStore Interface: The core abstraction defining operations for interacting with
//     a key-value store. All implementations share this common interface, allowing
//     applications to switch between different storage backends without code changes.
//     The interface methods return custom Error types that provide detailed information
//     about operation results.
//
//   - Error System: A structured error reporting mechanism using typed error codes
//     and descriptive messages. This system allows applications to make informed
//     decisions based on specific error conditions rather than generic errors.
//
//   - DBFactory: A function type that abstracts the creation of underlying db.KVDB
//     instances, providing dependency injection and flexible configuration of
//     storage backends.
//
// Implementations:
//
//	The package includes two implementations of the IStore interface:
//
//	- Local Store (lstore): A simple, non-distributed implementation that directly
//	  utilizes a db.KVDB instance. It manages write index progression internally
//	  using atomic operations to ensure thread safety. This implementation is suitable
//	  for single-node applications where distributed consensus is not required.
//	  Available in the "github.com/ValentinKolb/dKV/lib/store/lstore" package.
//
//	- Distributed Store (dstore): A implementation built on the Dragonboat
//	  RAFT consensus library. It distributes storage operations across multiple nodes
//	  with strong consistency guarantees. This implementation is appropriate for
//	  multi-node deployments requiring fault tolerance and high availability.
//	  Available in the "github.com/ValentinKolb/dKV/lib/store/dstore" package.
//
// This interface-driven approach allows applications to:
//   - Switch between local and distributed storage depending on deployment requirements
//   - Handle errors in a consistent and type-safe manner across implementations
//   - Abstract storage implementation details from application logic
package store
