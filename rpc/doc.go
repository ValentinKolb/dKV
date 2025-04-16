// Package rpc provides a comprehensive framework for remote procedure calls
// in the distributed key-value store system. It acts as the communication layer
// between clients and servers, enabling operations across network boundaries.
//
// The package is organized into several subpackages:
//
//   - common: Core data structures and utilities used across the RPC system,
//     including the Message protocol, configuration structures, and logging.
//
//   - transport: Network communication abstractions with pluggable implementations
//     (TCP, Unix sockets, HTTP).
//
//   - serializer: Message serialization with multiple format options (Binary, JSON, GOB)
//     for converting between Message objects and byte arrays.
//
//   - client: RPC client implementations for the store and lock manager interfaces,
//     allowing applications to interact with remote services transparently.
//
//   - server: RPC server components that handle incoming requests, including
//     adapters for store and lock manager operations.
package rpc
