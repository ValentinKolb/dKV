package unix

import (
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

// --------------------------------------------------------------------------
// Client Transport Factory Method
// --------------------------------------------------------------------------

// NewUnixClientTransport creates a new Unix client transport
func NewUnixClientTransport() transport.IRPCClientTransport {
	return base.NewBaseClientTransport(&clientConnector{})
}
