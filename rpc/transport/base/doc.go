// Package base provides a foundation for transport layers in the distributed key-value store,
// implementing core functionality for RPC communication independent of the specific
// network protocol (TCP, Unix sockets, etc.). It serves as a base layer that can be
// extended with protocol-specific connectors.
//
// The package focuses on:
//   - Protocol-agnostic client and server transport implementations
//   - Performance optimization through connection pooling and buffer reuse
//   - Frame-based message protocol with shardID and requestID tracking
//   - Automatic request routing and response correlation
//   - Robust error handling with retries and reconnection logic
//
// Key Components:
//
//   - IClientConnector/IServerConnector: Interfaces for protocol-specific operations
//     that allow extending the base transport with different network protocols.
//
//   - clientTransport: Core client implementation that manages multiple connections
//     with round-robin load balancing. Supports multiple connections per endpoint
//     for improved throughput.
//
//   - serverTransport: Core server implementation that accepts connections and
//     routes requests to the appropriate handler based on shardID.
//
// Performance Optimizations:
//
//   - Connection Pooling: Multiple connections per endpoint improve throughput
//     for high-load scenarios. This is particularly beneficial for large messages
//     where connection saturation becomes a bottleneck. For small messages (< 1KB),
//     a single connection per endpoint may actually perform better due to reduced
//     overhead.
//
//   - Buffer Pooling: The server uses a sync.Pool to reuse buffers, reducing
//     GC pressure and memory allocations.
//
//   - Asynchronous Processing: The client sends requests and correlates responses
//     asynchronously using unique request IDs, enabling higher throughput.
//
//   - Frame Batching: The transport uses net.Buffers to reduce syscalls when
//     writing frames, combining header and payload into a single write operation.
//
// Thread Safety:
//
//	All public methods are thread-safe. The client transport uses atomic operations
//	and mutexes to ensure concurrent access safety, while the server creates a
//	dedicated goroutine for each connection.
package base
