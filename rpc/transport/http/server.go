package http

import (
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/lni/dragonboat/v4/logger"
	"io"
	"net/http"
	"strconv"
	"time"
)

var Logger = logger.GetLogger("transport/rpc")

func NewHttpServerTransport() transport.IRPCServerTransport {
	return &httpServerTransport{}
}

type httpServerTransport struct {
	handler transport.ServerHandleFunc
	config  common.ServerConfig
}

// --------------------------------------------------------------------------
// Interface Methods (docu see transport.IRPCServerTransport)
// --------------------------------------------------------------------------

func (t *httpServerTransport) RegisterHandler(handler transport.ServerHandleFunc) {
	t.handler = handler
}

func (t *httpServerTransport) Listen(config common.ServerConfig) error {
	t.config = config

	// Create a new HTTP server
	mux := http.NewServeMux()

	// Register handler
	if t.config.LogLevel == "debug" {
		mux.HandleFunc("POST /{shardId}", loggerMiddleware(t.handleRequest))
	} else {
		mux.HandleFunc("POST /{shardId}", t.handleRequest)
	}

	// Get address from util
	Logger.Infof("Starting HTTP server on %s", t.config.Transport.Endpoint)

	// Set up the server with the address and handler
	return http.ListenAndServe(t.config.Transport.Endpoint, mux)
}

// --------------------------------------------------------------------------
// Helper Methods
// --------------------------------------------------------------------------

// handleRequest handles incoming HTTP requests and writes the response to the writer
func (t *httpServerTransport) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Parse shardId from request
	shardId, err := strconv.ParseUint(
		r.PathValue("shardId"),
		10, 64,
	)

	// Check if shardId is valid
	if err != nil {
		http.Error(w, "Invalid shardId", http.StatusBadRequest)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Check if body could be read
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Send the handler
	resp := t.handler(shardId, body)

	// Write response
	if _, err = w.Write(resp); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// --------------------------------------------------------------------------
// Middleware (logging)
// --------------------------------------------------------------------------

// responseWriter is a custom ResponseWriter that captures status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// writeHeader captures the status code before writing it
func (rw *responseWriter) writeHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// loggerMiddleware is a middleware that logs HTTP requests
func loggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create custom response writer to capture status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(rw, r)

		// Log the request
		duration := time.Since(start)
		Logger.Debugf("%s %s => %d took %s", r.Method, r.URL.Path, rw.statusCode, duration)
	}
}
