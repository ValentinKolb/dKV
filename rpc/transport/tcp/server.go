package tcp

import (
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/ValentinKolb/dKV/rpc/transport/base"
	"net"
)

const (
	defaultBufferSize = 512 * 1024 // 512 KB
)

// serverConnector implements the IServerConnector interface for TCP sockets
type serverConnector struct{}

// --------------------------------------------------------------------------
// Interface Methods (docu see base.IServerConnector)
// --------------------------------------------------------------------------

func (c *serverConnector) GetName() string {
	return "tcp"
}

func (c *serverConnector) Listen(config common.ServerConfig) (net.Listener, error) {
	// Create TCP socket listener
	listener, err := net.Listen("tcp", config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP socket: %v", err)
	}

	return listener, nil
}

// --------------------------------------------------------------------------
// Server Transport Factory Method
// --------------------------------------------------------------------------

// NewTCPDefaultServerTransport creates a new TCP server transport with default buffer size
func NewTCPDefaultServerTransport(workers int) transport.IRPCServerTransport {
	return NewTCPServerTransport(defaultBufferSize, workers)
}

// NewTCPServerTransport creates a new TCP server transport with specified buffer size
func NewTCPServerTransport(bufferSize, workers int) transport.IRPCServerTransport {
	return base.NewBaseServerTransport(&serverConnector{}, bufferSize, workers)
}
