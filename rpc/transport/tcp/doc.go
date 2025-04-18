// Package tcp implements TCPConf socket-based transport for the distributed key-value store's
// RPC system. It provides concrete implementations of the base package's connector
// interfaces optimized for TCPConf connections.
//
// This package builds on the base package's transport functionality, inheriting its
// performance optimizations including connection pooling, buffer reuse, and request
// routing. See the base package documentation for detailed information on the underlying
// transport mechanisms and performance characteristics.
//
// Key Components:
//
//   - clientConnector: TCPConf-specific implementation of base.IClientConnector
//
//   - serverConnector: TCPConf-specific implementation of base.IServerConnector
//
// The default server buffer size is set to 512 KB, which provides good performance
// for typical workloads, but can be customized for specific use cases.
package tcp
