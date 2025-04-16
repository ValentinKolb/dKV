// Package serializer provides message serialization capabilities for the distributed
// key-value store RPC system. It defines a common interface and multiple implementations
// for serializing and deserializing messages between client and server components.
//
// The package focuses on:
//   - Providing a consistent interface for different serialization formats
//   - Offering multiple implementations with different performance characteristics
//   - Supporting efficient encoding of the system's message structure
//   - Minimizing memory allocations and processing overhead
//
// Key Components:
//
//   - IRPCSerializer: Core interface that all serializer implementations must satisfy.
//
//   - binarySerializerImpl: Custom binary format implementation optimized for speed
//     and space efficiency. Uses a flag-based approach to encode only present fields,
//     resulting in compact serialized data with minimal overhead.
//
//   - gobSerializerImpl: Implementation using Go's built-in gob encoding, offering
//     good compatibility with Go's type system but with larger serialized sizes.
//
//   - jsonSerializerImpl: Implementation using JSON encoding, useful for debugging
//     or interoperability with other systems, but with lower performance.
//
// Performance Characteristics (based on benchmarks across various message types):
//
//   - Binary: Delivers superior performance with the smallest payload size. Highly optimized
//     for the application's specific message structure and recommended for production use.
//
//   - JSON: Offers acceptable performance with moderate payload sizes. Provides human-readable
//     output beneficial for debugging and system integration scenarios.
//
//   - GOB: Performs significantly worse than other implementations with consistently larger
//     payload sizes. Not recommended for use in this system as it provides no advantages
//     over Binary or JSON serialization.
//
// Thread Safety:
//
//	All serializer implementations are stateless and safe for concurrent use
//	across multiple goroutines without additional synchronization.
//
// Usage:
//
//	Serializers are typically created once and reused throughout the application:
//
//	  serializer := serializer.NewBinarySerializer()
//	  data, err := serializer.Serialize(message)
//	  // ... send data ...
//	  var receivedMsg common.Message
//	  err = serializer.Deserialize(receivedData, &receivedMsg)
package serializer
