package tcp

import (
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/ValentinKolb/dKV/rpc/transport/base"
	"net"
	"time"
)

const (
	defaultBufferSize = 512 * 1024 // 512 KB
)

// serverConnector implements the IServerConnector interface for TCPConf sockets
type serverConnector struct{}

// --------------------------------------------------------------------------
// Interface Methods (docu see base.IServerConnector)
// --------------------------------------------------------------------------

func (c *serverConnector) GetName() string {
	return "tcp"
}

func (c *serverConnector) Listen(config common.ServerConfig) (net.Listener, error) {
	// Create TCPConf socket listener
	listener, err := net.Listen("tcp", config.Transport.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCPConf socket: %v", err)
	}

	return listener, nil
}

// UpgradeConnection applies performance optimizations to a TCPConf connection
// using configuration values from TCPConf and SocketConf
func (c *serverConnector) UpgradeConnection(conn net.Conn, config common.ServerConfig) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil // Not a TCPConf connection, nothing to upgrade
	}

	// Apply TCPConf-specific settings
	// Disable Nagle's algorithm (TCPNoDelay) if configured
	if err := tcpConn.SetNoDelay(config.Transport.TCPNoDelay); err != nil {
		return err
	}

	// Set socket write buffer size if configured
	if config.Transport.WriteBufferSize > 0 {
		if err := tcpConn.SetWriteBuffer(config.Transport.WriteBufferSize); err != nil {
			return err
		}
	}

	// Set socket read buffer size if configured
	if config.Transport.ReadBufferSize > 0 {
		if err := tcpConn.SetReadBuffer(config.Transport.ReadBufferSize); err != nil {
			return err
		}
	}

	// Enable TCPConf keep-alive if configured
	if config.Transport.TCPKeepAliveSec > 0 {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			return err
		}

		// Set keep-alive period
		keepAlivePeriod := time.Duration(config.Transport.TCPKeepAliveSec) * time.Second
		if err := tcpConn.SetKeepAlivePeriod(keepAlivePeriod); err != nil {
			return err
		}
	}

	// Set TCPConf linger option if configured
	if config.Transport.TCPLingerSec >= 0 {
		if err := tcpConn.SetLinger(config.Transport.TCPLingerSec); err != nil {
			return err
		}
	}

	return nil
}

// --------------------------------------------------------------------------
// Server Transport Factory Method
// --------------------------------------------------------------------------

// NewTCPServerTransport creates a new TCPConf server transport with specified buffer size
func NewTCPServerTransport() transport.IRPCServerTransport {
	return base.NewBaseServerTransport(&serverConnector{})
}
