// Package common provides core data structures and utilities shared across
// the distributed key-value store system. It defines fundamental types,
// configuration structures, and protocol elements used by other packages.
//
// The package focuses on:
//   - Message protocol definition for inter-component communication
//   - Configuration structures for client and server components
//   - Custom logging implementation integrated with Dragonboat
//   - Utilities for Dragonboat (RAFT) integration
//
// Key Components:
//
//   - Message: Core data structure for all RPC communication between components,
//     with a flexible structure that adapts to different operation types.
//     Includes factory methods for creating various request and response messages.
//
//   - MessageType: Enumeration defining all supported operation types in the
//     system, categorized into key-value operations, lock operations, and
//     control messages.
//
//   - ServerConfig: Comprehensive configuration for server nodes, including
//     RAFT parameters, storage settings, network configuration, and operation modes.
//     Provides utilities for converting to Dragonboat-specific configurations.
//
//   - ClientConfig: Configuration for client components, controlling connection
//     parameters, timeouts, and retry behavior.
//
//   - Logger: Custom logging implementation that integrates with Dragonboat's
//     logging system while providing consistent formatting across the application.
package common
