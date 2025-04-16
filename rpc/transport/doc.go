// Package transport defines the interfaces and abstractions for RPC communication
// in the distributed key-value store. It provides a common contract that all transport
// implementations must fulfill, enabling protocol-agnostic communication.
//
// The package focuses on:
//   - Defining clear interfaces for client and server transport layers
//   - Supporting shard-based request routing
//   - Enabling multiple transport implementations (HTTP, TCP, Unix sockets)
//
// Key Components:
//
//   - IRPCClientTransport: Interface for client-side transport implementations that
//     handles connection management and request sending.
//
//   - IRPCServerTransport: Interface for server-side transport implementations that
//     receives requests and routes them to appropriate handlers.
//
//   - ServerHandleFunc: Function type for request handling callbacks.
package transport
