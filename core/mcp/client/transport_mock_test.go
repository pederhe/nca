package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pederhe/nca/core/mcp/common"
	"github.com/stretchr/testify/assert"
)

// MockTransport implements the Transport interface for testing
type MockTransport struct {
	messageHandler func(common.JSONRPCMessage)
	errorHandler   func(error)
	closeHandler   func()

	// For simulating responses
	responseQueue []common.JSONRPCMessage

	// For tracking sent messages
	sentMessages []common.JSONRPCMessage

	// Mock state
	isConnected bool
	mutex       sync.Mutex
	sessionID   string
}

// NewMockTransport creates a new mock transport for testing
func NewMockTransport() *MockTransport {
	return &MockTransport{
		responseQueue: make([]common.JSONRPCMessage, 0),
		sentMessages:  make([]common.JSONRPCMessage, 0),
		sessionID:     "mock-session-id",
		isConnected:   false,
	}
}

// Start simulates starting the transport - it does nothing for the mock
func (t *MockTransport) Start(ctx context.Context) error {
	fmt.Println("[DEBUG] Transport Start called")
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.isConnected = true
	fmt.Println("[DEBUG] Transport Start completed")
	return nil
}

// Send records the sent message and immediately processes the next response
func (t *MockTransport) Send(msg common.JSONRPCMessage) error {
	fmt.Printf("[DEBUG] Transport Send called: %v\n", msg)
	t.mutex.Lock()

	// Record the sent message
	t.sentMessages = append(t.sentMessages, msg)

	// Check if we have a response ready and a handler is set
	if len(t.responseQueue) > 0 && t.messageHandler != nil {
		response := t.responseQueue[0]
		t.responseQueue = t.responseQueue[1:]

		fmt.Printf("[DEBUG] Found message in response queue: %v\n", response)

		// For requests with an id, set the correct response id
		if id, exists := msg["id"]; exists {
			if respID, hasID := response["id"]; hasID {
				// If the response ID is zero or nil, use the request ID
				if respID == nil || respID == float64(0) {
					response["id"] = id
					fmt.Printf("[DEBUG] Updated response ID to: %v\n", id)
				}
			} else {
				// If the response doesn't have an ID, add one matching the request
				response["id"] = id
				fmt.Printf("[DEBUG] Added response ID: %v\n", id)
			}
		} else {
			// This is a notification without an ID
			// No need to set an ID on the response, but we still need to process it
			fmt.Printf("[DEBUG] Processing notification, no ID needed\n")
		}

		// Keep a reference to the message handler while locked
		handler := t.messageHandler

		// Unlock before calling the handler to avoid deadlocks
		t.mutex.Unlock()

		fmt.Printf("[DEBUG] Calling message handler\n")
		// Call the handler directly - this is synchronous, not in a goroutine
		handler(response)
		fmt.Printf("[DEBUG] Message handler call completed\n")

		return nil
	}

	// If no response, just unlock and return
	fmt.Printf("[DEBUG] No response queue message or no message handler\n")
	t.mutex.Unlock()
	return nil
}

// Close simulates closing the transport
func (t *MockTransport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.isConnected {
		return nil
	}

	t.isConnected = false

	if t.closeHandler != nil {
		t.closeHandler()
	}

	return nil
}

// SetCloseHandler sets the callback for when the connection is closed
func (t *MockTransport) SetCloseHandler(handler func()) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.closeHandler = handler
}

// SetErrorHandler sets the callback for when an error occurs
func (t *MockTransport) SetErrorHandler(handler func(error)) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.errorHandler = handler
}

// SetMessageHandler sets the callback for when a message is received
func (t *MockTransport) SetMessageHandler(handler func(common.JSONRPCMessage)) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Wrap the handler to add debugging
	wrappedHandler := func(msg common.JSONRPCMessage) {
		fmt.Printf("[DEBUG] Message handler received message: %v\n", msg)
		handler(msg)
		fmt.Printf("[DEBUG] Message handler processing completed\n")
	}

	t.messageHandler = wrappedHandler
}

// SessionID returns the session ID
func (t *MockTransport) SessionID() string {
	return t.sessionID
}

// QueueResponse adds a response to be sent when a message is received
func (t *MockTransport) QueueResponse(response common.JSONRPCMessage) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.responseQueue = append(t.responseQueue, response)
}

// GetSentMessages returns all messages that have been sent
func (t *MockTransport) GetSentMessages() []common.JSONRPCMessage {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.sentMessages
}

// SendMessageToClient simulates receiving a message from the server
func (t *MockTransport) SendMessageToClient(msg common.JSONRPCMessage) {
	t.mutex.Lock()
	handler := t.messageHandler
	t.mutex.Unlock()

	if handler != nil {
		handler(msg)
	}
}

// CreateMockServerInitializeResponse creates a standard initialize response for testing
func CreateMockServerInitializeResponse(id interface{}) common.JSONRPCMessage {
	return common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]interface{}{
			"protocolVersion": "1.0.0",
			"serverInfo": map[string]interface{}{
				"name":    "MockServer",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{
				"completion": true,
				"ping":       true,
				"logging":    true,
			},
			"instructions": "Mock server instructions",
		},
	}
}

// CreateMockResponseFromRequest creates a success response based on a request message
func CreateMockResponseFromRequest(request common.JSONRPCMessage, result interface{}) common.JSONRPCMessage {
	id, _ := request["id"]

	return common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}

// CreateMockErrorResponseFromRequest creates an error response based on a request message
func CreateMockErrorResponseFromRequest(request common.JSONRPCMessage, code int, message string) common.JSONRPCMessage {
	id, _ := request["id"]

	return common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
}

// DeserializeRequest unmarshals a JSONRPCMessage into a Request struct
func DeserializeRequest(msg common.JSONRPCMessage) (*common.Request, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	var request struct {
		JSONRPC string                 `json:"jsonrpc"`
		ID      interface{}            `json:"id"`
		Method  string                 `json:"method"`
		Params  map[string]interface{} `json:"params,omitempty"`
	}

	if err := json.Unmarshal(data, &request); err != nil {
		return nil, err
	}

	return &common.Request{
		Method: request.Method,
		Params: request.Params,
	}, nil
}

func TestMockTransportNotification(t *testing.T) {
	fmt.Println("[DEBUG] Starting TestMockTransportNotification test")

	// Create a mock transport
	transport := NewMockTransport()

	// Prepare responses
	responseWithID := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  "ok response with id",
	}
	transport.QueueResponse(responseWithID)

	responseWithoutID := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"result":  "ok response without id",
	}
	transport.QueueResponse(responseWithoutID)

	// Track received messages
	var receivedMessages []common.JSONRPCMessage

	// Set message handler
	transport.SetMessageHandler(func(msg common.JSONRPCMessage) {
		fmt.Printf("[DEBUG] Test received message: %v\n", msg)
		receivedMessages = append(receivedMessages, msg)
	})

	// Start transport
	err := transport.Start(context.Background())
	assert.NoError(t, err, "Start should not error")

	// Send request with ID
	requestWithID := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "test",
	}
	err = transport.Send(requestWithID)
	assert.NoError(t, err, "Send request with ID should not error")

	// Send notification without ID
	notificationWithoutID := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"method":  "notification",
	}
	err = transport.Send(notificationWithoutID)
	assert.NoError(t, err, "Send notification without ID should not error")

	// Verify received messages
	assert.Equal(t, 2, len(receivedMessages), "Should have received 2 messages")

	// Verify first response is for request with ID
	assert.NotNil(t, receivedMessages[0]["id"], "First response should have an ID")
	assert.Equal(t, "ok response with id", receivedMessages[0]["result"], "First response should have correct result")

	// Verify second response is for notification without ID
	assert.Equal(t, "ok response without id", receivedMessages[1]["result"], "Second response should have correct result")

	fmt.Println("[DEBUG] TestMockTransportNotification test completed")
}

func TestMockTransportBlockingCheck(t *testing.T) {
	fmt.Println("[DEBUG] Starting TestMockTransportBlockingCheck test")

	// Create a mock transport
	transport := NewMockTransport()

	// Prepare response queue
	transport.QueueResponse(common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"dummy": "response",
		},
	})

	transport.QueueResponse(common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"dummy": "notification response",
		},
	})

	// Set message handler
	messagesCh := make(chan common.JSONRPCMessage, 2)
	transport.SetMessageHandler(func(msg common.JSONRPCMessage) {
		fmt.Printf("[DEBUG] Test received message: %v\n", msg)
		messagesCh <- msg
	})

	// Start transport
	err := transport.Start(context.Background())
	assert.NoError(t, err, "Start should not error")

	// Send two messages
	fmt.Println("[DEBUG] Sending request message")
	requestMsg := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "test/method",
	}
	err = transport.Send(requestMsg)
	assert.NoError(t, err, "Send request should not error")

	// Wait for first response
	select {
	case response := <-messagesCh:
		fmt.Printf("[DEBUG] Received first response: %v\n", response)
		assert.NotNil(t, response["id"], "Response should have ID")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("[DEBUG] Failed to receive first response within timeout")
	}

	fmt.Println("[DEBUG] Sending notification message")
	notificationMsg := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"method":  "test/notification",
	}
	err = transport.Send(notificationMsg)
	assert.NoError(t, err, "Send notification should not error")

	// Wait for second response
	select {
	case response := <-messagesCh:
		fmt.Printf("[DEBUG] Received second response: %v\n", response)
		assert.Nil(t, response["id"], "Notification response should not have ID")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("[DEBUG] Failed to receive second response within timeout")
	}

	fmt.Println("[DEBUG] TestMockTransportBlockingCheck test completed")
}
