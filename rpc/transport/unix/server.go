package unix

import (
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/ValentinKolb/dKV/rpc/transport/base"
	"net"
	"os"
)

const (
	defaultBufferSize = 64 * 1024 // 64 KB
)

// serverConnector implements the IServerConnector interface for Unix sockets
type serverConnector struct{}

// --------------------------------------------------------------------------
// Interface Methods (docu see base.IServerConnector)
// --------------------------------------------------------------------------

func (c *serverConnector) GetName() string {
	return "unix"
}

func (c *serverConnector) Listen(config common.ServerConfig) (net.Listener, error) {
	socketPath := config.Endpoint

	// Remove existing socket file if it exists
	if err := os.RemoveAll(socketPath); err != nil {
		return nil, fmt.Errorf("failed to remove existing socket: %v", err)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Unix socket: %v", err)
	}

	return listener, nil
}

// --------------------------------------------------------------------------
// Server Transport Factory Method
// --------------------------------------------------------------------------

// NewUnixDefaultServerTransport creates a new Unix server transport with default buffer size
func NewUnixDefaultServerTransport() transport.IRPCServerTransport {
	return NewUnixServerTransport(defaultBufferSize)
}

// NewUnixServerTransport creates a new Unix server transport with specified buffer size
func NewUnixServerTransport(bufferSize uint64) transport.IRPCServerTransport {
	return base.NewBaseServerTransport(&serverConnector{}, bufferSize)
}
