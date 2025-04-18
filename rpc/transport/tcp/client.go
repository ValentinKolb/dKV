package tcp

import (
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/ValentinKolb/dKV/rpc/transport/base"
	"net"
	"time"
)

// clientConnector implements the IClientConnector interface for TCP sockets
type clientConnector struct{}

// --------------------------------------------------------------------------
// Interface Methods (docu see base.IClientConnector)
// --------------------------------------------------------------------------

func (c *clientConnector) GetName() string {
	return "tcp"
}

func (c *clientConnector) Connect(endpoint string) (net.Conn, error) {
	return net.Dial("tcp", endpoint)
}

// UpgradeConnection applies performance optimizations to a TCP connection
// using configuration values from TCPTransportConfig and SocketTransportConfig
func (c *clientConnector) UpgradeConnection(conn net.Conn, config common.ClientConfig) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil // Not a TCP connection, nothing to upgrade
	}

	// Apply TCP-specific settings
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

	// Enable TCP keep-alive if configured
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

	// Set TCP linger option if configured
	if config.Transport.TCPLingerSec >= 0 {
		if err := tcpConn.SetLinger(config.Transport.TCPLingerSec); err != nil {
			return err
		}
	}

	return nil
}

// --------------------------------------------------------------------------
// Client Transport Factory Method
// --------------------------------------------------------------------------

// NewTCPClientTransport creates a new TCP client transport
func NewTCPClientTransport() transport.IRPCClientTransport {
	return base.NewBaseClientTransport(&clientConnector{})
}
