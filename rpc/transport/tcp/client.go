package tcp

import (
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/ValentinKolb/dKV/rpc/transport/base"
	"net"
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

// --------------------------------------------------------------------------
// Client Transport Factory Method
// --------------------------------------------------------------------------

// NewTCPClientTransport creates a new TCP client transport
func NewTCPClientTransport() transport.IRPCClientTransport {
	return base.NewBaseClientTransport(&clientConnector{})
}
