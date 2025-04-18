package unix

import (
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/ValentinKolb/dKV/rpc/transport/base"
	"net"
)

// clientConnector implements the IClientConnector interface for Unix sockets
type clientConnector struct{}

// --------------------------------------------------------------------------
// Interface Methods (docu see base.IClientConnector)
// --------------------------------------------------------------------------

func (c *clientConnector) GetName() string {
	return "unix"
}

func (c *clientConnector) Connect(endpoint string) (net.Conn, error) {
	return net.Dial("unix", endpoint)
}

// UpgradeConnection applies performance optimizations to a Unix socket connection
func (c *clientConnector) UpgradeConnection(conn net.Conn, config common.ClientConfig) error {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return nil // Not a Unix connection, nothing to upgrade
	}

	sendBufferSize := 128 * 1024 // 128 KB for client send
	if err := unixConn.SetWriteBuffer(sendBufferSize); err != nil {
		return err
	}

	recvBufferSize := 256 * 1024 // 256 KB for client receive
	if err := unixConn.SetReadBuffer(recvBufferSize); err != nil {
		return err
	}

	return nil
}

// --------------------------------------------------------------------------
// Client Transport Factory Method
// --------------------------------------------------------------------------

// NewUnixClientTransport creates a new Unix client transport
func NewUnixClientTransport() transport.IRPCClientTransport {
	return base.NewBaseClientTransport(&clientConnector{})
}
