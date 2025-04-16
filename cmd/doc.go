// Package cmd implements the command-line interface for the dKV distributed
// key-value store. It provides a hierarchical command structure with operations
// for running the server and interacting with it as a client.
//
// The package is organized into several subpackages:
//
//   - kv: Commands for key-value store operations (get, set, delete, etc.)
//   - lock: Commands for locking operations (acquire, release)
//   - serve: Commands for starting and configuring the dKV server
//   - util: Shared utilities for command-line processing and configuration (internal use)
//
// See dkv -help for a list of all commands.
package cmd
