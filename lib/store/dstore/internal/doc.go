// Package internal provides the communication protocol structures and serialization
// logic for the dstore package. It defines the wire format used to transmit operations
// between the store client and the distributed state machine.
//
// This package is intended for internal use by the dstore implementation and should
// not be imported directly by external code.
//
// The package consists of two main components:
//
//   - Command System: Defines write operations (Set, SetE, etc.) that modify the
//     state of the database. Commands are serialized and proposed to the RAFT cluster,
//     executed on the state machine, and produce results that are returned to the client.
//     The Command structure includes efficient binary serialization.
//
//   - Query System: Defines read operations (Get, Has, etc.) that retrieve data from
//     the database without modifying its state. Queries are executed locally on the
//     statemachine and therefore do not require serialization.
//
// Protocol Design:
//
//	The Command serialization format is optimized for:
//
//	- Minimal Size: Commands use a compact binary encoding that minimizes the amount
//	  of data transmitted over the network and stored in the RAFT log.
//
//	- Efficient Parsing: The format is designed for fast serialization and deserialization
//	  with minimal allocations.
//
// Command Format:
//
//	Commands are serialized into a binary format with the following structure:
//
//	- 1 byte: Command type (Set, SetE, SetEIfUnset, Expire, Delete)
//	- 8 bytes: ExpireIn value (uint64, big endian)
//	- 8 bytes: DeleteIn value (uint64, big endian)
//	- 4 bytes: Key length (uint32, big endian)
//	- N bytes: Key data (string as byte array)
//	- M bytes: Value data (optional, only present for Set-type operations)
//
//	This format ensures efficient storage in the RAFT log while providing all
//	necessary information for the operation.
//
// Query Format:
//
//	Queries use a simpler structure as they are not persisted in the RAFT log:
//
//	- Type: The query operation to perform (Get, Has, GetDBInfo)
//	- Key: The key to query (empty for operations like GetDBInfo)
//
// Type Mapping:
//
//	The package provides bidirectional mapping between:
//	- Command types and db.Feature (db.KVDB) flags for feature detection
//	- String representations for logging and debugging
//
// Thread Safety:
//
//	The types in this package are not thread-safe and should not be shared
//	across goroutines without external synchronization. However, this is not
//	typically an issue as the RAFT protocol ensures sequential processing of
//	commands on the state machine.
package internal
