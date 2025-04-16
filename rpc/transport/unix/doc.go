// Package unix implements a transport layer for the distributed key-value store's
// RPC system using Unix domain sockets. It provides optimized communication for
// processes running on the same machine.
//
// This package extends the base transport layer with Unix socket-specific connectors
// while inheriting all core functionality like connection pooling, request routing,
// and error handling from the base package.
//
// Key Components:
//
//   - clientConnector: Establishes connections using Unix domain sockets
//
//   - serverConnector: Creates Unix socket listeners and accepts connections
//
// Performance Characteristics:
//
//   - Default buffer size: 64 KB, optimized for local communication patterns
//   - Reduced overhead: Eliminates TCP/IP stack processing for better performance
//   - Lower latency: Direct kernel-mediated IPC avoids network subsystem overhead
package unix
