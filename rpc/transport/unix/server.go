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
	socketPath := config.Transport.Endpoint

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

// UpgradeConnection applies performance optimizations to a Unix socket connection
func (c *serverConnector) UpgradeConnection(conn net.Conn, config common.ServerConfig) error {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return nil // Not a Unix connection, nothing to upgrade
	}

	sendBufferSize := 256 * 1024 // 256 KB for server send
	if err := unixConn.SetWriteBuffer(sendBufferSize); err != nil {
		return err
	}

	recvBufferSize := 256 * 1024 // 256 KB for server receive
	if err := unixConn.SetReadBuffer(recvBufferSize); err != nil {
		return err
	}

	return nil
}

// --------------------------------------------------------------------------
// Server Transport Factory Method
// --------------------------------------------------------------------------

// NewUnixServerTransport creates a new Unix server transport with specified buffer size
func NewUnixServerTransport() transport.IRPCServerTransport {
	return base.NewBaseServerTransport(&serverConnector{})
}
