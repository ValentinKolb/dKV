package transport

import (
	"github.com/ValentinKolb/dKV/rpc/common"
)

// --------------------------------------------------------------------------
// Server Transport
// --------------------------------------------------------------------------

// ServerHandleFunc is a function type that handles incoming requests
// This function is called by a server transport layer when a request is received
// It takes a shardId and a request as parameters and returns a response
type ServerHandleFunc func(shardId uint64, req []byte) (resp []byte)

// IRPCServerTransport is the interface for the RPC transport layer
// It must accept a RPCServerConfig as a parameter
type IRPCServerTransport interface {
	// RegisterHandler registers a handler for the transport layer
	// This handler should be called when a request is received
	// The transport layer is responsible for routing the request to the appropriate shard
	RegisterHandler(handler ServerHandleFunc)
	// Listen starts the transport layer and listens for incoming requests
	Listen(config common.ServerConfig) error
}

// --------------------------------------------------------------------------
// Client Transport
// --------------------------------------------------------------------------

// IRPCClientTransport is the interface for the RPC client transport
type IRPCClientTransport interface {
	// Connect initializes the transport with the given configuration
	Connect(config common.ClientConfig) error
	// Send sends a request to the server and returns the response
	Send(shardId uint64, req []byte) (resp []byte, err error)
	// Close closes the transport connection
	Close() error
}
