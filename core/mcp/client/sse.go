package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pederhe/nca/core/mcp/common"
)

// SseError represents an error in the SSE connection
type SseError struct {
	Code    int
	Message string
	Cause   error
}

// Error implements the error interface
func (e *SseError) Error() string {
	msg := fmt.Sprintf("SSE error: %s", e.Message)
	if e.Code != 0 {
		msg = fmt.Sprintf("%s (code %d)", msg, e.Code)
	}
	if e.Cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Cause)
	}
	return msg
}

// UnauthorizedError represents an unauthorized error
type UnauthorizedError struct {
	Message string
}

// Error implements the error interface
func (e *UnauthorizedError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("Unauthorized: %s", e.Message)
	}
	return "Unauthorized"
}

// OAuthProvider is an interface that provides OAuth authentication services
type OAuthProvider interface {
	// GetToken retrieves the access token
	GetToken() (string, error)

	// RefreshToken refreshes the access token
	RefreshToken() (string, error)
}

// SSEClientTransportOptions configures options for SSEClientTransport
type SSEClientTransportOptions struct {
	// AuthProvider is the OAuth client provider used for authentication
	AuthProvider OAuthProvider

	// HttpClient specifies the HTTP client to use
	// If nil, http.DefaultClient will be used
	HttpClient *http.Client

	// RequestHeaders are additional HTTP headers to apply to each request
	RequestHeaders http.Header
}

// SSEClientTransport implements a client transport based on Server-Sent Events
// It will use Server-Sent Events to receive messages and use separate POST requests to send messages
type SSEClientTransport struct {
	url          *url.URL
	endpoint     *url.URL
	eventSource  *eventSource
	httpClient   *http.Client
	authProvider OAuthProvider
	reqHeaders   http.Header

	closeHandler   func()
	errorHandler   func(error)
	messageHandler func(common.JSONRPCMessage)
	sessionID      string

	mutex       sync.RWMutex
	isConnected bool

	ctx    context.Context
	cancel context.CancelFunc

	endpointChan chan struct{}
}

// eventSource is a simplified implementation of EventSource
type eventSource struct {
	url         *url.URL
	httpClient  *http.Client
	headers     http.Header
	messageChan chan *eventSourceMessage
	errorChan   chan error
	readyChan   chan struct{}
	closeChan   chan struct{}
	response    *http.Response
	reader      *bufio.Reader
	mutex       sync.RWMutex
	closed      bool
}

// eventSourceMessage represents messages received from EventSource
type eventSourceMessage struct {
	Event string
	Data  string
	ID    string
}

// NewSSEClientTransport creates a new SSE client transport
func NewSSEClientTransport(url *url.URL, opts *SSEClientTransportOptions) *SSEClientTransport {
	var httpClient *http.Client
	var authProvider OAuthProvider
	var headers http.Header

	if opts != nil {
		if opts.HttpClient != nil {
			httpClient = opts.HttpClient
		}
		authProvider = opts.AuthProvider
		headers = opts.RequestHeaders
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	if headers == nil {
		headers = make(http.Header)
	}

	return &SSEClientTransport{
		url:          url,
		httpClient:   httpClient,
		authProvider: authProvider,
		reqHeaders:   headers,
	}
}

// Start starts the SSE connection
func (t *SSEClientTransport) Start(ctx context.Context) error {
	t.mutex.Lock()

	if t.isConnected {
		t.mutex.Unlock()
		return errors.New("SSEClientTransport is already started! If using the Client class, note that Connect() will automatically call Start()")
	}

	t.ctx, t.cancel = context.WithCancel(ctx)
	t.isConnected = true
	t.mutex.Unlock()

	return t.startOrAuth()
}

// startOrAuth starts the SSE connection or performs authentication
func (t *SSEClientTransport) startOrAuth() error {
	// Clean up previous resources
	if t.eventSource != nil {
		t.eventSource.close()
		t.eventSource = nil
	}

	// Create headers with authentication information
	headers := t.createHeaders()
	headers.Set("Accept", "text/event-stream")

	// Create EventSource
	es := &eventSource{
		url:         t.url,
		httpClient:  t.httpClient,
		headers:     headers,
		messageChan: make(chan *eventSourceMessage),
		errorChan:   make(chan error),
		readyChan:   make(chan struct{}),
		closeChan:   make(chan struct{}),
	}

	// Start EventSource
	go es.connect(t.ctx)

	// Wait for connection to be ready or error
	select {
	case <-es.readyChan:
		// Connection successful
		t.eventSource = es
		t.endpointChan = make(chan struct{})

		// Start processing messages
		go t.processEvents()

		// Wait for endpoint event or timeout
		select {
		case <-t.endpointChan:
			return nil
		case <-t.ctx.Done():
			return t.ctx.Err()
		case <-time.After(30 * time.Second): // Increase timeout to 30 seconds
			return fmt.Errorf("waiting for endpoint event timeout")
		}

	case err := <-es.errorChan:
		// If it's a 401 error, try refreshing the token and reconnect
		var sseErr *SseError
		if errors.As(err, &sseErr) && sseErr.Code == http.StatusUnauthorized && t.authProvider != nil {
			if _, err := t.authProvider.RefreshToken(); err != nil {
				return &UnauthorizedError{Message: "refresh token failed"}
			}
			return t.startOrAuth()
		}

		// Set isConnected to false when connection fails
		t.mutex.Lock()
		t.isConnected = false
		t.mutex.Unlock()

		return err

	case <-t.ctx.Done():
		return t.ctx.Err()
	}
}

// createHeaders creates HTTP headers with authentication information
func (t *SSEClientTransport) createHeaders() http.Header {
	headers := t.reqHeaders.Clone()

	// If there's an authentication provider, add authentication headers
	if t.authProvider != nil {
		token, err := t.authProvider.GetToken()
		if err == nil && token != "" {
			headers.Set("Authorization", "Bearer "+token)
		}
	}

	return headers
}

// processEvents processes events received from EventSource
func (t *SSEClientTransport) processEvents() {
	for {
		select {
		case msg := <-t.eventSource.messageChan:
			switch msg.Event {
			case "message":
				// Default message event
				t.handleMessage(msg.Data)

			case "endpoint":
				// Process endpoint event
				endpointURL, err := url.Parse(msg.Data)
				if err != nil {
					if t.errorHandler != nil {
						t.errorHandler(fmt.Errorf("failed to parse endpoint URL: %w", err))
					}
					continue
				}

				// Ensure endpoint URL has the same origin as the connection URL
				if endpointURL.Scheme == "" || endpointURL.Host == "" {
					// This is a relative URL, resolve it based on the connection URL
					endpointURL = t.url.ResolveReference(endpointURL)
				} else if endpointURL.Scheme != t.url.Scheme || endpointURL.Host != t.url.Host {
					if t.errorHandler != nil {
						t.errorHandler(fmt.Errorf("endpoint origin does not match connection origin: %s", endpointURL.String()))
					}
					continue
				}

				t.mutex.Lock()
				t.endpoint = endpointURL
				t.mutex.Unlock()

				// Close channel outside lock
				if t.endpointChan != nil {
					close(t.endpointChan)
					t.endpointChan = nil
				}
			}

		case err := <-t.eventSource.errorChan:
			// Process error
			if t.errorHandler != nil {
				t.errorHandler(err)
			}

			// Close connection
			t.Close()
			return

		case <-t.eventSource.closeChan:
			// EventSource closed
			t.handleClose()
			return

		case <-t.ctx.Done():
			// Context canceled
			t.Close()
			return
		}
	}
}

// handleMessage processes message data
func (t *SSEClientTransport) handleMessage(data string) {
	var message common.JSONRPCMessage
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		if t.errorHandler != nil {
			t.errorHandler(fmt.Errorf("JSON parsing error: %w", err))
		}
		return
	}

	if t.messageHandler != nil {
		t.messageHandler(message)
	}
}

// Close closes the SSE connection
func (t *SSEClientTransport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.cancel != nil {
		t.cancel()
	}

	if t.eventSource != nil {
		t.eventSource.close()
		t.eventSource = nil
	}

	t.isConnected = false

	return nil
}

// Send sends a JSON-RPC message
func (t *SSEClientTransport) Send(msg common.JSONRPCMessage) error {
	t.mutex.RLock()
	endpoint := t.endpoint
	t.mutex.RUnlock()

	if endpoint == nil {
		return errors.New("not connected")
	}

	// Serialize message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("JSON serialization error: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(t.ctx, "POST", endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	headers := t.createHeaders()
	for k, v := range headers {
		req.Header[k] = v
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		if t.errorHandler != nil {
			t.errorHandler(err)
		}
		return err
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// If it's a 401 error, try refreshing the token and resend
		if resp.StatusCode == http.StatusUnauthorized && t.authProvider != nil {
			if _, err := t.authProvider.RefreshToken(); err != nil {
				return &UnauthorizedError{Message: "refresh token failed"}
			}

			// Recursive call, try resending
			return t.Send(msg)
		}

		// Read error information
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST request error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// SetCloseHandler sets the callback for connection closure
func (t *SSEClientTransport) SetCloseHandler(handler func()) {
	t.closeHandler = handler
}

// SetErrorHandler sets the callback for error handling
func (t *SSEClientTransport) SetErrorHandler(handler func(error)) {
	t.errorHandler = handler
}

// SetMessageHandler sets the callback for message reception
func (t *SSEClientTransport) SetMessageHandler(handler func(common.JSONRPCMessage)) {
	t.messageHandler = handler
}

// SessionID returns the session ID
func (t *SSEClientTransport) SessionID() string {
	return t.sessionID
}

// handleClose processes connection closure event
func (t *SSEClientTransport) handleClose() {
	t.mutex.Lock()
	t.isConnected = false
	t.mutex.Unlock()

	if t.closeHandler != nil {
		t.closeHandler()
	}
}

// connect connects to EventSource
func (es *eventSource) connect(ctx context.Context) {
	es.mutex.Lock()
	if es.closed {
		es.mutex.Unlock()
		return
	}
	es.mutex.Unlock()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", es.url.String(), nil)
	if err != nil {
		es.sendError(err)
		return
	}

	// Set headers
	for k, v := range es.headers {
		req.Header[k] = v
	}

	// Send request
	resp, err := es.httpClient.Do(req)
	if err != nil {
		es.sendError(err)
		return
	}

	// Check response
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		es.sendError(&SseError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("HTTP error: %s", resp.Status),
		})
		return
	}

	// Check Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		resp.Body.Close()
		es.sendError(&SseError{
			Message: fmt.Sprintf("Invalid Content-Type: %s", contentType),
		})
		return
	}

	// Save response
	es.mutex.Lock()
	es.response = resp
	es.reader = bufio.NewReader(resp.Body)
	es.mutex.Unlock()

	// Notify connection ready
	close(es.readyChan)

	// Start reading events
	go es.readEvents(ctx)
}

// readEvents reads events from the event stream
func (es *eventSource) readEvents(ctx context.Context) {
	defer func() {
		es.mutex.Lock()
		if es.response != nil {
			es.response.Body.Close()
			es.response = nil
		}
		es.mutex.Unlock()

		close(es.closeChan)
	}()

	var event, data, id string

	for {
		select {
		case <-ctx.Done():
			return

		default:
			es.mutex.RLock()
			reader := es.reader
			es.mutex.RUnlock()

			if reader == nil {
				return
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					es.sendError(err)
				}
				return
			}

			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")

			if line == "" {
				// Empty line indicates event end
				if data != "" {
					es.messageChan <- &eventSourceMessage{
						Event: event,
						Data:  data,
						ID:    id,
					}
					event = ""
					data = ""
					// id is not reset, it persists until the next id field
				}
				continue
			}

			if strings.HasPrefix(line, ":") {
				// Comment line, ignore
				continue
			}

			colonIndex := strings.Index(line, ":")
			if colonIndex == -1 {
				// No colon, field name is entire line, value is empty
				field := line

				switch field {
				case "event":
					event = ""
				case "data":
					if data == "" {
						data = ""
					} else {
						data += "\n"
					}
				case "id":
					id = ""
				}
				continue
			}

			field := line[:colonIndex]
			var value string
			if colonIndex+1 < len(line) {
				// Skip space after colon
				if line[colonIndex+1] == ' ' {
					value = line[colonIndex+2:]
				} else {
					value = line[colonIndex+1:]
				}
			}

			switch field {
			case "event":
				event = value
			case "data":
				if data == "" {
					data = value
				} else {
					data += "\n" + value
				}
			case "id":
				id = value
			}
		}
	}
}

// sendError sends error
func (es *eventSource) sendError(err error) {
	select {
	case es.errorChan <- err:
	default:
		// Error channel is full or closed
	}
}

// close closes EventSource
func (es *eventSource) close() {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	if es.closed {
		return
	}

	es.closed = true

	if es.response != nil {
		es.response.Body.Close()
		es.response = nil
	}
}
