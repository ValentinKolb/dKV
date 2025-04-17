package base

import (
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/client"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/puzpuzpuz/xsync/v3"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var Logger = logger.GetLogger("transport/rpc")

// -----------------------------------------------------------
// Interface Definitions for dependency injection
// -----------------------------------------------------------

// IClientConnector defines the interface for transport-specific connection operations
type IClientConnector interface {
	// Connect establishes a single connection based on the provided configuration
	Connect(endpoint string) (net.Conn, error)

	// GetName returns the name of the transport type (e.g., "unix", "tcp")
	GetName() string
}

// -----------------------------------------------------------
// Helper Types
// -----------------------------------------------------------

// responseResult contains the result of a request
type responseResult struct {
	data []byte
	err  error
}

// clientConnection represents a single net connection
type clientConnection struct {
	conn         net.Conn
	endpoint     string
	stopCh       chan struct{} // Close signal for the reader goroutine
	requestChans *xsync.MapOf[uint64, chan responseResult]
	connMu       sync.Mutex // Protects the connection itself
	parent       *clientTransport
}

// clientTransport implements the core client transport functionality
// independent of the specific transport medium (unix, tcp, etc.)
type clientTransport struct {
	connector     IClientConnector
	config        common.ClientConfig
	connections   []*clientConnection
	connectionsMu sync.RWMutex
	nextConnIndex uint64 // Atomic counter for Round Robin
	nextRequestID uint64 // Atomic counter for unique request IDs
	stopping      bool   // Signals shutdown
}

// -----------------------------------------------------------
// Transport Factory Method (used for tcp, unix, etc.)
// -----------------------------------------------------------

// NewBaseClientTransport creates a new base client transport with the specified connector
func NewBaseClientTransport(connector IClientConnector) transport.IRPCClientTransport {
	return &clientTransport{
		connector:     connector,
		nextRequestID: 1, // Start from 1
	}
}

// --------------------------------------------------------------------------
// Interface Methods (docu see transport.IRPCClientTransport)
// --------------------------------------------------------------------------

/*

TODO clean Send method up, better retire logic eg backoff

TODO server retires for senden responses ??

*/

func (t *clientTransport) Connect(config common.ClientConfig) error {
	if len(config.Endpoints) == 0 {
		return fmt.Errorf("no endpoints provided")
	}

	// Store the util
	t.config = config
	t.stopping = false

	// Close all existing connections
	t.closeConnections()

	// Set default value for ConnectionsPerEndpoint
	connectionsPerEP := 1
	if config.ConnectionsPerEndpoint > 0 {
		connectionsPerEP = config.ConnectionsPerEndpoint
	}

	// Create connections
	t.connections = make([]*clientConnection, 0, len(config.Endpoints)*connectionsPerEP)

	// Initialize client connections
	for _, endpoint := range config.Endpoints {
		// Create multiple connections per endpoint
		for i := 0; i < connectionsPerEP; i++ {
			// Connect to the endpoint
			conn, err := t.connector.Connect(endpoint)
			if err != nil {
				Logger.Warningf("Failed to connect to %s (connection %d/%d): %v", endpoint, i+1, connectionsPerEP, err)
				continue
			}

			// Create a new client connection
			clientConn := &clientConnection{
				conn:         conn,
				endpoint:     endpoint,
				stopCh:       make(chan struct{}),
				requestChans: xsync.NewMapOf[uint64, chan responseResult](),
				parent:       t,
			}

			// Add to our connections list
			t.connectionsMu.Lock()
			t.connections = append(t.connections, clientConn)
			t.connectionsMu.Unlock()

			Logger.Infof("Connected to %s (connection %d/%d)", endpoint, i+1, connectionsPerEP)

			// Start the response reader
			go clientConn.readResponses()
		}
	}

	// Check if we have at least one connection
	if len(t.connections) == 0 {
		return fmt.Errorf("failed to connect to any endpoint")
	}

	Logger.Infof("Connected to %d out of %d connections to %d endpoints using %s transport",
		len(t.connections), len(config.Endpoints)*connectionsPerEP, len(config.Endpoints), t.connector.GetName())

	return nil
}

func (t *clientTransport) Send(shardId uint64, req []byte) (resp []byte, err error) {
	// Generate a unique request ID
	requestID := atomic.AddUint64(&t.nextRequestID, 1)

	// Select a connection via Round Robin
	conn := t.getNextConnection()
	if conn == nil {
		return nil, fmt.Errorf("no active connections available")
	}

	// Create a channel for the response
	respCh := make(chan responseResult, 1)

	// Register the request
	conn.requestChans.Store(requestID, respCh)

	// Clean up when done
	defer func() {
		conn.requestChans.Delete(requestID)
	}()

	send := func(connection *clientConnection) ([]byte, error) {
		// Test if connection is still valid
		if connection.conn == nil {
			return nil, fmt.Errorf("connection is closed")
		}

		// Set write timeout
		if t.config.TimeoutSecond > 0 {
			timeout := time.Duration(t.config.TimeoutSecond) * time.Second
			connection.conn.SetWriteDeadline(time.Now().Add(timeout))
		}

		// Lock the connection only for writing
		connection.connMu.Lock()
		//Logger.Infof("send %d", requestID)
		err := writeFrame(connection.conn, shardId, requestID, req)
		connection.connMu.Unlock()

		if err != nil {
			return nil, err
		}

		// Wait for response or timeout
		var timeoutCh <-chan time.Time
		if t.config.TimeoutSecond > 0 {
			timeout := time.Duration(t.config.TimeoutSecond) * time.Second
			timeoutCh = time.After(timeout)
		} else {
			timeoutCh = make(chan time.Time) // Never triggers
		}

		select {
		case result := <-respCh:
			//Logger.Infof("recv %d", requestID)
			return result.data, result.err
		case <-timeoutCh:
			return nil, fmt.Errorf("request timed out")
		}
	}

	// First try with the selected connection
	data, err := send(conn)
	if err == nil {
		return data, nil
	}

	// On error: Try other connections
	var lastErr = err
	for i := 1; i < t.config.RetryCount; i++ {
		// For connection errors, try to use another connection
		conn = t.getNextConnection()
		if conn == nil {
			// No more connections available
			return nil, fmt.Errorf("no active connections available after %d retries", i)
		}

		// Try with the new connection
		data, err := send(conn)
		if err == nil {
			return data, nil
		}
		lastErr = err

		// Wait briefly before the next attempt
		time.Sleep(time.Millisecond * 100)
	}

	// All attempts failed
	return nil, fmt.Errorf("failed to send request after %d retries: %v", t.config.RetryCount, lastErr)
}

func (t *clientTransport) Close() error {
	t.stopping = true
	t.closeConnections()
	return nil
}

// --------------------------------------------------------------------------
// Helper Methods
// --------------------------------------------------------------------------

// getNextConnection selects the next connection via Round Robin
func (t *clientTransport) getNextConnection() *clientConnection {
	t.connectionsMu.RLock()
	defer t.connectionsMu.RUnlock()

	if len(t.connections) == 0 {
		return nil
	}

	// Simple Round Robin algorithm
	var index uint64
	if len(t.connections) == 1 {
		// optimize for single connection
		index = 0
	} else {
		index = atomic.AddUint64(&t.nextConnIndex, 1) % uint64(len(t.connections))
	}
	return t.connections[index]
}

// closeConnections closes all active connections
func (t *clientTransport) closeConnections() {
	t.connectionsMu.Lock()
	defer t.connectionsMu.Unlock()

	for _, conn := range t.connections {
		// Signal reader goroutine to stop
		close(conn.stopCh)

		// Close the connection
		if conn.conn != nil {
			conn.conn.Close()
		}
	}

	// Empty the list
	t.connections = nil
}

// readResponses reads responses in a loop and distributes them to waiting requests
func (c *clientConnection) readResponses() {
	for {
		// Check if we should stop
		select {
		case <-c.stopCh:
			return
		default:
			// Continue
		}

		// Set timeout if configured
		if c.parent.config.TimeoutSecond > 0 {
			timeout := time.Duration(c.parent.config.TimeoutSecond) * time.Second
			c.conn.SetReadDeadline(time.Now().Add(timeout))
		}

		// Read the response frame
		shardID, requestID, data, err := readFrame(c.conn, nil)

		// Find the corresponding request channel
		respCh, found := c.requestChans.Load(requestID)

		if found {
			if err != nil {
				// Send the error to the waiting request
				respCh <- responseResult{nil, fmt.Errorf("error reading response: %v", err)}
			} else {
				// Send the response to the waiting request
				respCh <- responseResult{data, nil}
			}
		} else if err != nil {
			// Error with unknown request ID
			client.Logger.Errorf("Error reading response with unknown request ID %d: %v", requestID, err)

			// Try to restore the connection
			if err := c.reconnect(); err != nil {
				client.Logger.Errorf("Failed to reconnect to %s: %v", c.endpoint, err)
				return
			}
		} else {
			// Warning for unknown request ID
			client.Logger.Warningf("Received response for unknown request ID %d with shard ID %d", requestID, shardID)
		}
	}
}

// reconnect restores a connection with the connector
func (c *clientConnection) reconnect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	// Close the old connection
	if c.conn != nil {
		c.conn.Close()
	}

	// Connect
	conn, err := c.parent.connector.Connect(c.endpoint)

	if err != nil {
		c.conn = conn
	}
	return err
}
