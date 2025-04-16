// Package http implements an HTTP-based transport layer for RPC communication
// in the distributed key-value store system. It provides concrete implementations
// of the transport interfaces defined in the parent package, enabling communication
// between clients and servers over HTTP.
//
// The package focuses on:
//   - Client-side HTTP transport for sending RPC requests to servers
//   - Server-side HTTP transport for receiving and handling RPC requests
//   - Round-robin load balancing across multiple server endpoints
//   - Request routing based on shard IDs
//
// Key Components:
//
//   - httpClientTransport: Implements IRPCClientTransport interface, managing
//     connections to server endpoints, handling request routing, and implementing
//     retry mechanisms. It uses round-robin selection for load balancing across
//     multiple server endpoints.
//
//   - httpServerTransport: Implements IRPCServerTransport interface, setting up
//     an HTTP server that routes incoming requests to the appropriate handler
//     based on the shard ID specified in the URL path.
//
// Thread Safety:
//
//	The client transport is thread-safe and can be used concurrently. It uses
//	atomic operations for the round-robin counter to ensure thread safety when
//	selecting server endpoints.
//
// This implementation offers several advantages:
//   - Simple integration with existing HTTP infrastructure
//   - Built-in load balancing across multiple server endpoints
//   - Straightforward error handling and retry mechanisms
//   - Logging middleware for request monitoring
package http
