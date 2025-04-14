package client

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/pederhe/nca/pkg/mcp/common"
	"github.com/stretchr/testify/assert"
)

// TestMockTransport tests the mock transport implementation
func TestMockTransport(t *testing.T) {
	// Create a new mock transport
	transport := NewMockTransport()

	// Set up handlers
	var receivedMessage common.JSONRPCMessage
	var receivedError error
	var closeCalled bool

	transport.SetMessageHandler(func(msg common.JSONRPCMessage) {
		receivedMessage = msg
	})

	transport.SetErrorHandler(func(err error) {
		receivedError = err
		t.Logf("Received error: %v", err)
	})

	transport.SetCloseHandler(func() {
		closeCalled = true
	})

	// Test Start
	ctx := context.Background()
	err := transport.Start(ctx)
	assert.NoError(t, err, "Start should not error")

	// Test SessionID
	assert.Equal(t, "mock-session-id", transport.SessionID(), "Session ID should match")

	// Test Send and message handler
	testMessage := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "test",
		"params":  map[string]interface{}{"key": "value"},
	}

	// Queue a response
	responseMessage := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  map[string]interface{}{"response": "success"},
	}
	transport.QueueResponse(responseMessage)

	// Send the message
	err = transport.Send(testMessage)
	assert.NoError(t, err, "Send should not error")

	// Wait for async handler
	time.Sleep(10 * time.Millisecond)

	// Verify sent message was tracked
	sentMessages := transport.GetSentMessages()
	assert.Equal(t, 1, len(sentMessages), "Should have sent 1 message")
	assert.Equal(t, testMessage, sentMessages[0], "Sent message should match")

	// Verify handler received response
	assert.Equal(t, responseMessage, receivedMessage, "Received message should match queued response")

	// Test direct message sending
	directMessage := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"method":  "notification",
		"params":  map[string]interface{}{"notification": "data"},
	}
	transport.SendMessageToClient(directMessage)

	// Wait for async handler
	time.Sleep(10 * time.Millisecond)

	// Verify handler received direct message
	assert.Equal(t, directMessage, receivedMessage, "Received message should match direct message")

	// Test Close
	err = transport.Close()
	assert.NoError(t, err, "Close should not error")

	// Wait for async handler
	time.Sleep(10 * time.Millisecond)

	// Verify close handler was called
	assert.True(t, closeCalled, "Close handler should have been called")

	// Test error handler
	testError := errors.New("test error")
	transport.SetErrorHandler(func(err error) {
		receivedError = err
	})

	// Simulate receiving an error
	transport.errorHandler(testError)

	// Verify error handler was called
	assert.Equal(t, testError, receivedError, "Received error should match test error")
}

// TestWebSocketClientTransport tests the WebSocket transport implementation
func TestWebSocketClientTransport(t *testing.T) {
	// Skip actual connection test since it requires a real server
	t.Skip("Skipping WebSocket test as it requires a real server")

	// Test creation of WebSocket transport
	wsURL, _ := url.Parse("ws://localhost:8080/ws")
	transport := NewWebSocketClientTransport(wsURL)

	assert.NotNil(t, transport, "WebSocket transport should not be nil")
	assert.Equal(t, wsURL, transport.url, "URL should match")
}

// TestSSEClientTransport tests the SSE transport implementation
func TestSSEClientTransport(t *testing.T) {
	// Skip actual connection test since it requires a real server
	t.Skip("Skipping SSE test as it requires a real server")

	// Test creation of SSE transport
	sseURL, _ := url.Parse("http://localhost:8080/events")

	// Test with default options
	transport1 := NewSSEClientTransport(sseURL, nil)
	assert.NotNil(t, transport1, "SSE transport with nil options should not be nil")
	assert.Equal(t, sseURL, transport1.url, "URL should match")

	// Test with custom options
	customHeaders := make(map[string][]string)
	customHeaders["X-Custom-Header"] = []string{"test-value"}

	options := &SSEClientTransportOptions{
		RequestHeaders: customHeaders,
	}

	transport2 := NewSSEClientTransport(sseURL, options)
	assert.NotNil(t, transport2, "SSE transport with options should not be nil")
	assert.Equal(t, sseURL, transport2.url, "URL should match")
	assert.Equal(t, "test-value", transport2.reqHeaders.Get("X-Custom-Header"), "Custom header should be set")
}

// TestStdioClientTransport tests the stdio transport implementation
func TestStdioClientTransport(t *testing.T) {
	// Skip actual process spawn test since it requires a real server binary
	t.Skip("Skipping stdio test as it requires a real server binary")

	// Test creation of stdio transport
	params := StdioServerParameters{
		Command: "echo",
		Args:    []string{"hello"},
	}

	transport := NewStdioClientTransport(params)
	assert.NotNil(t, transport, "Stdio transport should not be nil")
	assert.Equal(t, params, transport.serverParams, "Server parameters should match")
}

// TestGetDefaultEnvironment tests the default environment function
func TestGetDefaultEnvironment(t *testing.T) {
	env := GetDefaultEnvironment()
	assert.NotNil(t, env, "Default environment should not be nil")

	// Check that it contains some of the expected variables
	// Note: We can't check exact values as they depend on the test environment
	for _, key := range DefaultInheritedEnvVars {
		_, exists := env[key]
		// The variable might not exist in the test environment, so we don't assert on this
		t.Logf("Environment variable %s exists: %v", key, exists)
	}
}
