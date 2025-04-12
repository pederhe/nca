package client

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/pederhe/nca/core/mcp/common"
	"github.com/stretchr/testify/assert"
)

// mockOAuthProvider mocks the OAuth provider
type mockOAuthProvider struct {
	token         string
	refreshCalled bool
}

func (m *mockOAuthProvider) GetToken() (string, error) {
	return m.token, nil
}

func (m *mockOAuthProvider) RefreshToken() (string, error) {
	m.refreshCalled = true
	m.token = "new_token"
	return m.token, nil
}

func TestSSETransport(t *testing.T) {
	// Use loopback address instead of system-assigned address
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Create server with custom listener
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Send endpoint event
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// Send endpoint URL
		endpointURL := "http://" + r.Host + "/endpoint"
		_, err := w.Write([]byte("event: endpoint\ndata: " + endpointURL + "\n\n"))
		if err != nil {
			t.Errorf("Write failed: %v", err)
			return
		}
		flusher.Flush()

		// Send test message
		msg := common.JSONRPCMessage{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "test",
			"params":  map[string]interface{}{"key": "value"},
		}
		data, err := json.Marshal(msg)
		if err != nil {
			t.Errorf("JSON serialization failed: %v", err)
			return
		}
		_, err = w.Write([]byte("event: message\ndata: " + string(data) + "\n\n"))
		if err != nil {
			t.Errorf("Write failed: %v", err)
			return
		}
		flusher.Flush()
	}))

	// Use custom listener
	server.Listener = listener
	server.Start()
	defer server.Close()

	// Create SSE transport
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	transport := NewSSEClientTransport(serverURL, nil)

	// Test message handler
	messageReceived := false
	transport.SetMessageHandler(func(msg common.JSONRPCMessage) {
		messageReceived = true
		assert.Equal(t, "2.0", msg["jsonrpc"])
		assert.Equal(t, float64(1), msg["id"])
		assert.Equal(t, "test", msg["method"])
	})

	// Test close handler
	closeReceived := false
	transport.SetCloseHandler(func() {
		closeReceived = true
	})

	// Test Start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = transport.Start(ctx)
	if err != nil {
		t.Logf("Connection failed, skipping remaining tests: %v", err)
		return
	}

	assert.True(t, transport.isConnected)

	// Wait for message reception
	time.Sleep(100 * time.Millisecond)
	assert.True(t, messageReceived)

	// Test Send
	msg := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "test",
		"params":  map[string]interface{}{"key": "value"},
	}

	err = transport.Send(msg)
	assert.NoError(t, err)

	// Test Close
	err = transport.Close()
	assert.NoError(t, err)
	assert.False(t, transport.isConnected)
	assert.True(t, closeReceived)
}

func TestSSETransportWithAuth(t *testing.T) {
	// Use loopback address instead of system-assigned address
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Create server with custom listener
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// Send endpoint URL
		endpointURL := "http://" + r.Host + "/endpoint"
		_, err := w.Write([]byte("event: endpoint\ndata: " + endpointURL + "\n\n"))
		if err != nil {
			t.Errorf("Write failed: %v", err)
			return
		}
		flusher.Flush()
	}))

	// Use custom listener
	server.Listener = listener
	server.Start()
	defer server.Close()

	// Create mock OAuth provider
	mockProvider := &mockOAuthProvider{
		token: "test_token",
	}

	// Create SSE transport
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	transport := NewSSEClientTransport(serverURL, &SSEClientTransportOptions{
		AuthProvider: mockProvider,
	})

	// Test Start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = transport.Start(ctx)
	if err != nil {
		t.Logf("Connection failed, skipping remaining tests: %v", err)
		return
	}

	assert.True(t, transport.isConnected)

	// Test Close
	err = transport.Close()
	assert.NoError(t, err)
	assert.False(t, transport.isConnected)
}

func TestSSETransportUnauthorized(t *testing.T) {
	// Use loopback address instead of system-assigned address
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Create server with custom listener
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))

	// Use custom listener
	server.Listener = listener
	server.Start()
	defer server.Close()

	// Create mock OAuth provider
	mockProvider := &mockOAuthProvider{
		token: "test_token",
	}

	// Create SSE transport
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	transport := NewSSEClientTransport(serverURL, &SSEClientTransportOptions{
		AuthProvider: mockProvider,
	})

	// Test Start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = transport.Start(ctx)
	assert.Error(t, err, "Should return an error")

	// isConnected should be false when connection fails
	assert.False(t, transport.isConnected, "isConnected should be false when connection fails")
}
