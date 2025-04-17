package base

import (
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"io"
	"math"
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
	connector         IServerConnector
	handler           transport.ServerHandleFunc
	config            common.ServerConfig
	listener          net.Listener
	bufferPool        *sync.Pool
	bufferSize        int
	maxWorkersPerConn int
}

// -----------------------------------------------------------
// Transport Factory Method (used for tcp, unix, etc.)
// -----------------------------------------------------------

// NewBaseServerTransport creates a new base server transport with per-connection worker pool
func NewBaseServerTransport(connector IServerConnector, bufferSize int, maxWorkersPerConn int) transport.IRPCServerTransport {

	// minimum one worker per connection
	maxWorkersPerConn = int(math.Min(float64(maxWorkersPerConn), 1))

	return &serverTransport{
		connector:         connector,
		bufferSize:        bufferSize,
		maxWorkersPerConn: maxWorkersPerConn,
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

	Logger.Infof("Starting %s server on %s with %d workers per connection",
		t.connector.GetName(), config.Endpoint, t.maxWorkersPerConn)

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

	// Create a semaphore to limit concurrent workers for this connection
	// The buffered channel acts as a counting semaphore
	workerSemaphore := make(chan struct{}, t.maxWorkersPerConn)

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup

	// Create a mutex to protect writes to the connection
	var connMutex sync.Mutex

	// Handler function that processes requests in worker goroutines
	handleResponse := func(shardID, requestID uint64, data []byte) {
		// When done, release the semaphore and mark worker as done
		defer func() {
			<-workerSemaphore // Release semaphore slot
			wg.Done()         // Mark worker as done
		}()

		// Process the request
		start := time.Now()
		resp := t.handler(shardID, data)
		Logger.Debugf("Processed request for shard %d with requestID %d took %s", shardID, requestID, time.Since(start))

		// Protect writes to the connection with a mutex
		connMutex.Lock()
		defer connMutex.Unlock()

		if timeout > 0 {
			if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
				Logger.Errorf("Failed to set write deadline: %v", err)
				return
			}
		}

		// Write the response with the same requestID
		if err := writeFrame(conn, shardID, requestID, resp); err != nil {
			Logger.Errorf("Failed to write response: %v", err)
		}
	}

	// Function to handle incoming requests
	handleRequest := func() error {
		if timeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
				return fmt.Errorf("failed to set read deadline: %v", err)
			}
		}

		// Get a buffer from the pool
		buf := t.bufferPool.Get().([]byte)

		// Read the frame with requestID
		shardID, requestID, data, err := readFrame(conn, buf)

		// Error reading frame
		if err != nil {
			t.bufferPool.Put(buf)
			return err
		}

		// Acquire a slot in the semaphore (blocks if maxWorkersPerConn is reached)
		// This is the key mechanism that limits the number of concurrent workers
		workerSemaphore <- struct{}{}

		// Increment the wait group counter
		wg.Add(1)

		// Process in a goroutine
		go func() {
			defer t.bufferPool.Put(buf)
			handleResponse(shardID, requestID, data)
		}()

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

	// Wait for all workers to finish before closing the connection
	// This ensures we don't lose any in-progress work
	wg.Wait()
}
