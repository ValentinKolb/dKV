package base

import (
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/client"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/puzpuzpuz/xsync/v3"
	"math/rand"
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

	// UpgradeConnection applies protocol-specific settings to an established connection
	UpgradeConnection(conn net.Conn, config common.ClientConfig) error
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

func (t *clientTransport) Connect(config common.ClientConfig) error {
	if len(config.Transport.Endpoints) == 0 {
		return fmt.Errorf("no endpoints provided")
	}

	// Store the config
	t.config = config
	t.stopping = false

	// Close all existing connections
	t.closeConnections()

	// Set default value for ConnectionsPerEndpoint
	connectionsPerEP := 1
	if config.Transport.ConnectionsPerEndpoint > 0 {
		connectionsPerEP = config.Transport.ConnectionsPerEndpoint
	}

	// Create connections
	t.connections = make([]*clientConnection, 0, len(config.Transport.Endpoints)*connectionsPerEP)

	// Initialize client connections
	for _, endpoint := range config.Transport.Endpoints {
		// Create multiple connections per endpoint
		for i := 0; i < connectionsPerEP; i++ {
			clientConn := &clientConnection{
				conn:         nil, // Will be set by reconnect
				endpoint:     endpoint,
				stopCh:       make(chan struct{}),
				requestChans: xsync.NewMapOf[uint64, chan responseResult](),
				parent:       t,
			}

			// Establish the initial connection using reconnect
			if err := clientConn.reconnect(); err != nil {
				Logger.Warningf("Failed to connect to %s (connection %d/%d): %v", endpoint, i+1, connectionsPerEP, err)
				continue
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
		len(t.connections), len(config.Transport.Endpoints)*connectionsPerEP, len(config.Transport.Endpoints), t.connector.GetName())

	return nil
}

func (t *clientTransport) Send(shardId uint64, req []byte) (resp []byte, err error) {
	// Generate a unique request ID
	requestID := atomic.AddUint64(&t.nextRequestID, 1)

	// Define the send function to be used in retries
	send := func(connection *clientConnection) ([]byte, error) {
		// Test if connection is still valid
		if connection.conn == nil {
			return nil, fmt.Errorf("connection is closed")
		}

		// Create a channel for the response
		respCh := make(chan responseResult, 1)

		// Register the request
		connection.requestChans.Store(requestID, respCh)

		// Ensure we clean up when done
		defer connection.requestChans.Delete(requestID)

		// Set write timeout
		if t.config.TimeoutSecond > 0 {
			timeout := time.Duration(t.config.TimeoutSecond) * time.Second
			connection.conn.SetWriteDeadline(time.Now().Add(timeout))
		}

		// Lock the connection only for writing
		connection.connMu.Lock()
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
			return result.data, result.err
		case <-timeoutCh:
			return nil, fmt.Errorf("request timed out")
		}
	}

	// Retry logic with exponential backoff
	var lastErr error

	// We always try at least once, and up to maxRetries times
	maxRetries := t.config.Transport.RetryCount
	if maxRetries < 1 {
		maxRetries = 1
	}

	// Initial backoff duration in milliseconds
	backoffMs := 50

	for i := 0; i < maxRetries; i++ {
		conn := t.getNextConnection()
		if conn == nil {
			return nil, fmt.Errorf("no active connections available")
		}

		// Try with this connection
		data, err := send(conn)
		if err == nil {
			return data, nil
		}

		lastErr = err
		Logger.Debugf("Request attempt %d/%d failed: %v", i+1, maxRetries, err)

		if i < maxRetries {
			// Exponential backoff with a small random jitter (+-10%)
			jitter := float64(backoffMs) * (0.9 + 0.2*rand.Float64())
			backoffDuration := time.Duration(jitter) * time.Millisecond
			time.Sleep(backoffDuration)
			backoffMs *= 2
		}
	}

	// All attempts failed
	return nil, fmt.Errorf("failed to send request after %d attempts: %v", t.config.Transport.RetryCount, lastErr)
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

// reconnect establishes or restores a connection to the endpoint
func (c *clientConnection) reconnect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	// Close the old connection if it exists
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// Connect to the endpoint
	conn, err := c.parent.connector.Connect(c.endpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", c.endpoint, err)
	}

	// Upgrade the connection with protocol-specific settings
	if err := c.parent.connector.UpgradeConnection(conn, c.parent.config); err != nil {
		conn.Close()
		return fmt.Errorf("failed to upgrade connection to %s: %v", c.endpoint, err)
	}

	c.conn = conn
	return nil
}
