package base

import (
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"io"
	"net"
	"sync"
	"time"
)

// -----------------------------------------------------------
// Interface Definitions for dependency injection
// -----------------------------------------------------------

// IServerConnector defines the interface for transport-specific server operations
type IServerConnector interface {
	// Listen creates a listener and returns it
	Listen(config common.ServerConfig) (net.Listener, error)

	// GetName returns the name of the transport type (e.g., "unix", "tcp")
	GetName() string
}

// -----------------------------------------------------------
// Helper Types
// -----------------------------------------------------------

// serverTransport implements the core server transport functionality
type serverTransport struct {
	connector  IServerConnector
	handler    transport.ServerHandleFunc
	config     common.ServerConfig
	listener   net.Listener
	bufferPool *sync.Pool
	bufferSize uint64
}

// -----------------------------------------------------------
// Transport Factory Method (used for tcp, unix, etc.)
// -----------------------------------------------------------

// NewBaseServerTransport creates a new base server transport
func NewBaseServerTransport(connector IServerConnector, bufferSize uint64) transport.IRPCServerTransport {
	return &serverTransport{
		connector:  connector,
		bufferSize: bufferSize,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
	}
}

// --------------------------------------------------------------------------
// Interface Methods (docu see transport.IRPCClientTransport)
// --------------------------------------------------------------------------

func (t *serverTransport) RegisterHandler(handler transport.ServerHandleFunc) {
	t.handler = handler
}

func (t *serverTransport) Listen(config common.ServerConfig) error {
	t.config = config

	// Create listener using the connector
	listener, err := t.connector.Listen(config)
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}
	t.listener = listener

	Logger.Infof("Starting %s server on %s", t.connector.GetName(), config.Endpoint)

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			Logger.Errorf("Accept error: %v", err)
			continue
		}

		// Handle the connection in a goroutine
		go t.handleConnection(conn)
	}
}

// --------------------------------------------------------------------------
// Helper Methods
// --------------------------------------------------------------------------

// handleConnection handles incoming requests for one connection
func (t *serverTransport) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Timeout in seconds
	timeout := time.Duration(t.config.TimeoutSecond) * time.Second

	// Function to handle requests
	handleRequest := func() error {
		if timeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
				return fmt.Errorf("failed to set read deadline: %v", err)
			}
		}

		// Get a buffer from the pool
		buf := t.bufferPool.Get().([]byte)

		// Put buffer back in pool
		defer t.bufferPool.Put(buf)

		// Read the frame with requestID
		shardID, requestID, data, err := readFrame(conn, buf)

		// Error reading frame
		if err != nil {
			return err
		}

		// Process the request
		start := time.Now()
		resp := t.handler(shardID, data)
		Logger.Infof("Processed request for shard %d with requestID %d took %s", shardID, requestID, time.Since(start))

		if timeout > 0 {
			if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
				return fmt.Errorf("failed to set write deadline: %v", err)
			}
		}

		// Write the response with the same requestID
		if err := writeFrame(conn, shardID, requestID, resp); err != nil {
			return err
		}

		return nil
	}

	// Handle requests in a loop
	for {
		// Handle request
		err := handleRequest()

		// Case EOF: Connection closed by client
		if err == io.EOF {
			Logger.Infof("Connection closed by client")
			break
		}

		// Case error: log and close connection
		if err != nil {
			Logger.Errorf("Error handling request: %v", err)
			break
		}
	}
}
