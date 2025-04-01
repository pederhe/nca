package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/pederhe/nca/core/mcp/common"
	"github.com/stretchr/testify/assert"
)

// mockOAuthProvider 模拟 OAuth 提供者
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
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置 SSE 头部
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// 发送 endpoint 事件
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// 发送 endpoint URL
		endpointURL := "http://" + r.Host + "/endpoint"
		_, err := w.Write([]byte("event: endpoint\ndata: " + endpointURL + "\n\n"))
		assert.NoError(t, err)
		flusher.Flush()

		// 发送测试消息
		msg := common.JSONRPCMessage{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "test",
			"params":  map[string]interface{}{"key": "value"},
		}
		data, err := json.Marshal(msg)
		assert.NoError(t, err)
		_, err = w.Write([]byte("event: message\ndata: " + string(data) + "\n\n"))
		assert.NoError(t, err)
		flusher.Flush()
	}))
	defer server.Close()

	// 创建 SSE 传输
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	transport := NewSSEClientTransport(serverURL, nil)

	// 测试消息处理器
	messageReceived := false
	transport.SetMessageHandler(func(msg common.JSONRPCMessage) {
		messageReceived = true
		assert.Equal(t, "2.0", msg["jsonrpc"])
		assert.Equal(t, float64(1), msg["id"])
		assert.Equal(t, "test", msg["method"])
	})

	// 测试关闭处理器
	closeReceived := false
	transport.SetCloseHandler(func() {
		closeReceived = true
	})

	// 测试 Start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = transport.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, transport.isConnected)

	// 等待消息接收
	time.Sleep(100 * time.Millisecond)
	assert.True(t, messageReceived)

	// 测试 Send
	msg := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "test",
		"params":  map[string]interface{}{"key": "value"},
	}

	err = transport.Send(msg)
	assert.NoError(t, err)

	// 测试 Close
	err = transport.Close()
	assert.NoError(t, err)
	assert.False(t, transport.isConnected)
	assert.True(t, closeReceived)
}

func TestSSETransportWithAuth(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证认证头
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 设置 SSE 头部
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// 发送 endpoint URL
		endpointURL := "http://" + r.Host + "/endpoint"
		_, err := w.Write([]byte("event: endpoint\ndata: " + endpointURL + "\n\n"))
		assert.NoError(t, err)
		flusher.Flush()
	}))
	defer server.Close()

	// 创建模拟的 OAuth 提供者
	mockProvider := &mockOAuthProvider{
		token: "test_token",
	}

	// 创建 SSE 传输
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	transport := NewSSEClientTransport(serverURL, &SSEClientTransportOptions{
		AuthProvider: mockProvider,
	})

	// 测试 Start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = transport.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, transport.isConnected)

	// 测试 Close
	err = transport.Close()
	assert.NoError(t, err)
	assert.False(t, transport.isConnected)
}

func TestSSETransportUnauthorized(t *testing.T) {
	// 创建测试服务器，返回 401 错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	// 创建模拟的 OAuth 提供者
	mockProvider := &mockOAuthProvider{
		token: "test_token",
	}

	// 创建 SSE 传输
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	transport := NewSSEClientTransport(serverURL, &SSEClientTransportOptions{
		AuthProvider: mockProvider,
	})

	// 测试 Start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = transport.Start(ctx)
	assert.Error(t, err)
	assert.IsType(t, &UnauthorizedError{}, err)
	assert.False(t, transport.isConnected)
}
