package http

import (
	"bytes"
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/client"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

func NewHttpClientTransport() transport.IRPCClientTransport {
	return &httpClientTransport{}
}

type httpClientTransport struct {
	serverURLs []*url.URL
	client     *http.Client
	counter    uint32
	retryCount int
}

// --------------------------------------------------------------------------
// Interface Methods (docu see transport.IRPCClientTransport)
// --------------------------------------------------------------------------

func (transport *httpClientTransport) Connect(config common.ClientConfig) error {
	// Parse each server URL
	parsedURLs := make([]*url.URL, len(config.Endpoints))
	for i, server := range config.Endpoints {
		parsedURL, err := url.Parse(server)
		if err != nil {
			return err
		}
		parsedURLs[i] = parsedURL
	}

	// Create client with default transport
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     time.Duration(config.TimeoutSecond) * time.Second,
		},
	}

	// Set the client and server URLs
	transport.client = client
	transport.serverURLs = parsedURLs
	transport.counter = 0
	transport.retryCount = config.RetryCount

	// No error
	return nil
}

func (transport *httpClientTransport) Send(shardId uint64, req []byte) (resp []byte, err error) {
	// Check if the transport is initialized
	if transport.client == nil {
		return nil, fmt.Errorf("http transport not initialized")
	}

	// Select the next server via round-robin
	idx := atomic.AddUint32(&transport.counter, 1) % uint32(len(transport.serverURLs))
	serverURL := transport.serverURLs[idx]

	// Create the complete URL
	requestURL := fmt.Sprintf("%s/%v", serverURL.String(), shardId)

	// Create the request
	httpRequest, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(req))
	if err != nil {
		return nil, err
	}

	// Send the request (with retries)
	var httpResponse *http.Response
	defer func() {
		if httpResponse != nil {
			if err := httpResponse.Body.Close(); err != nil {
				client.Logger.Errorf("Failed to close response body: %v", err)
			}
		}
	}()
	for i := 0; i < transport.retryCount; i++ {
		httpResponse, err = transport.client.Do(httpRequest)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	// Check if the response status code is OK
	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %s", httpResponse.Status)
	}

	// Read the response body
	return io.ReadAll(httpResponse.Body)
}

func (transport *httpClientTransport) Close() error {
	// Close the client
	if transport.client != nil {
		transport.client.CloseIdleConnections()
	}

	// Reset the client and server URLs
	transport.client = nil
	transport.serverURLs = nil

	return nil
}
