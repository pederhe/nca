package common

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// MockTransport implements the Transport interface for testing
type MockTransport struct {
	closeHandler    func()
	errorHandler    func(error)
	messageHandler  func(JSONRPCMessage)
	messages        []JSONRPCMessage
	closed          bool
	sessionID       string
	mu              sync.Mutex
	shouldFailSend  bool
	shouldFailStart bool
}

// NewMockTransport creates a new mock transport for testing
func NewMockTransport() *MockTransport {
	return &MockTransport{
		messages:  make([]JSONRPCMessage, 0),
		sessionID: "mock-session-id",
	}
}

// Start implements Transport interface
func (mt *MockTransport) Start(ctx context.Context) error {
	if mt.shouldFailStart {
		return errors.New("mock start failure")
	}
	return nil
}

// Send implements Transport interface
func (mt *MockTransport) Send(msg JSONRPCMessage) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if mt.closed {
		return errors.New("transport closed")
	}

	if mt.shouldFailSend {
		return errors.New("mock send failure")
	}

	mt.messages = append(mt.messages, msg)
	return nil
}

// Close implements Transport interface
func (mt *MockTransport) Close() error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if mt.closed {
		return errors.New("transport already closed")
	}

	mt.closed = true

	if mt.closeHandler != nil {
		mt.closeHandler()
	}

	return nil
}

// SetCloseHandler implements Transport interface
func (mt *MockTransport) SetCloseHandler(handler func()) {
	mt.closeHandler = handler
}

// SetErrorHandler implements Transport interface
func (mt *MockTransport) SetErrorHandler(handler func(error)) {
	mt.errorHandler = handler
}

// SetMessageHandler implements Transport interface
func (mt *MockTransport) SetMessageHandler(handler func(JSONRPCMessage)) {
	mt.messageHandler = handler
}

// SessionID implements Transport interface
func (mt *MockTransport) SessionID() string {
	return mt.sessionID
}

// SimulateReceiveMessage simulates receiving a message from the remote endpoint
func (mt *MockTransport) SimulateReceiveMessage(msg JSONRPCMessage) {
	if mt.messageHandler != nil {
		mt.messageHandler(msg)
	}
}

// SimulateError simulates an error in the transport
func (mt *MockTransport) SimulateError(err error) {
	if mt.errorHandler != nil {
		mt.errorHandler(err)
	}
}

// SimulateClose simulates the transport being closed by the remote endpoint
func (mt *MockTransport) SimulateClose() {
	mt.mu.Lock()
	mt.closed = true
	mt.mu.Unlock()

	if mt.closeHandler != nil {
		mt.closeHandler()
	}
}

// GetSentMessages returns all messages sent through this transport
func (mt *MockTransport) GetSentMessages() []JSONRPCMessage {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	return mt.messages
}

// SetShouldFailSend configures the transport to fail send operations
func (mt *MockTransport) SetShouldFailSend(shouldFail bool) {
	mt.shouldFailSend = shouldFail
}

// SetShouldFailStart configures the transport to fail start operations
func (mt *MockTransport) SetShouldFailStart(shouldFail bool) {
	mt.shouldFailStart = shouldFail
}

// TestTransportInterface tests the Transport interface definition
func TestTransportInterface(t *testing.T) {
	// Create a mock transport
	mt := NewMockTransport()

	// Verify it implements the Transport interface
	var _ Transport = mt

	// Test SessionID
	if id := mt.SessionID(); id != "mock-session-id" {
		t.Errorf("Expected session ID to be 'mock-session-id', got '%s'", id)
	}

	// Test SetMessageHandler
	messageReceived := false
	testMsg := JSONRPCMessage{"method": "test"}

	mt.SetMessageHandler(func(msg JSONRPCMessage) {
		messageReceived = true
		if msg["method"] != "test" {
			t.Errorf("Expected message method to be 'test', got '%v'", msg["method"])
		}
	})

	mt.SimulateReceiveMessage(testMsg)
	if !messageReceived {
		t.Error("Expected message handler to be called")
	}

	// Test SetErrorHandler
	errorReceived := false
	testErr := errors.New("test error")

	mt.SetErrorHandler(func(err error) {
		errorReceived = true
		if err.Error() != "test error" {
			t.Errorf("Expected error to be 'test error', got '%v'", err)
		}
	})

	mt.SimulateError(testErr)
	if !errorReceived {
		t.Error("Expected error handler to be called")
	}

	// Test SetCloseHandler
	closeCalled := false

	mt.SetCloseHandler(func() {
		closeCalled = true
	})

	mt.SimulateClose()
	if !closeCalled {
		t.Error("Expected close handler to be called")
	}

	// Test Send
	mt = NewMockTransport() // Create a new mock to reset closed state
	err := mt.Send(testMsg)
	if err != nil {
		t.Errorf("Expected no error from Send, got: %v", err)
	}

	messages := mt.GetSentMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 sent message, got %d", len(messages))
	}

	if messages[0]["method"] != "test" {
		t.Errorf("Expected sent message method to be 'test', got '%v'", messages[0]["method"])
	}

	// Test failure conditions
	mt.SetShouldFailSend(true)
	err = mt.Send(testMsg)
	if err == nil {
		t.Error("Expected error from Send when shouldFailSend is true, got nil")
	}

	// Test closed transport
	mt = NewMockTransport()
	mt.Close()
	err = mt.Send(testMsg)
	if err == nil {
		t.Error("Expected error from Send when transport is closed, got nil")
	}

	// Test close when already closed
	err = mt.Close()
	if err == nil {
		t.Error("Expected error from Close when transport is already closed, got nil")
	}

	// Test Start
	mt = NewMockTransport()
	err = mt.Start(context.Background())
	if err != nil {
		t.Errorf("Expected no error from Start, got: %v", err)
	}

	mt.SetShouldFailStart(true)
	err = mt.Start(context.Background())
	if err == nil {
		t.Error("Expected error from Start when shouldFailStart is true, got nil")
	}
}
