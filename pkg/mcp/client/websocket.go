package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pederhe/nca/pkg/mcp/common"
)

// WebSocketClientTransport implements a client transport based on the WebSocket protocol
type WebSocketClientTransport struct {
	socket         *websocket.Conn
	url            *url.URL
	closeHandler   func()
	errorHandler   func(error)
	messageHandler func(common.JSONRPCMessage)
	sessionID      string
	mutex          sync.Mutex
	isConnected    bool
}

// NewWebSocketClientTransport creates a new WebSocket client transport instance
func NewWebSocketClientTransport(url *url.URL) *WebSocketClientTransport {
	return &WebSocketClientTransport{
		url: url,
	}
}

// Start starts the WebSocket connection
func (t *WebSocketClientTransport) Start(ctx context.Context) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.isConnected {
		return errors.New("WebSocketClientTransport is already started! If using the Client class, note that Connect() will automatically call Start()")
	}

	headers := make(map[string][]string)
	dialer := websocket.Dialer{}

	var err error
	t.socket, _, err = dialer.DialContext(ctx, t.url.String(), headers)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}

	t.isConnected = true

	// Start receiving messages
	go t.receiveMessages()

	return nil
}

// receiveMessages handles incoming WebSocket messages
func (t *WebSocketClientTransport) receiveMessages() {
	for {
		_, message, err := t.socket.ReadMessage()
		if err != nil {
			if t.errorHandler != nil {
				t.errorHandler(fmt.Errorf("read WebSocket message error: %w", err))
			}
			t.handleClose()
			return
		}

		var rpcMessage common.JSONRPCMessage
		if err := json.Unmarshal(message, &rpcMessage); err != nil {
			if t.errorHandler != nil {
				t.errorHandler(fmt.Errorf("parse JSON-RPC message error: %w", err))
			}
			continue
		}

		if t.messageHandler != nil {
			t.messageHandler(rpcMessage)
		}
	}
}

// Close closes the WebSocket connection
func (t *WebSocketClientTransport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.isConnected {
		return nil
	}

	if t.socket != nil {
		err := t.socket.Close()
		if err != nil {
			return fmt.Errorf("close WebSocket connection error: %w", err)
		}
	}

	t.isConnected = false
	t.handleClose()

	return nil
}

// Send sends a JSON-RPC message through the WebSocket connection
func (t *WebSocketClientTransport) Send(msg common.JSONRPCMessage) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.isConnected || t.socket == nil {
		return errors.New("not connected")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("serialize message error: %w", err)
	}

	err = t.socket.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return fmt.Errorf("write WebSocket message error: %w", err)
	}

	return nil
}

// SetCloseHandler sets the connection close callback
func (t *WebSocketClientTransport) SetCloseHandler(handler func()) {
	t.closeHandler = handler
}

// SetErrorHandler sets the error handling callback
func (t *WebSocketClientTransport) SetErrorHandler(handler func(error)) {
	t.errorHandler = handler
}

// SetMessageHandler sets the message reception callback
func (t *WebSocketClientTransport) SetMessageHandler(handler func(common.JSONRPCMessage)) {
	t.messageHandler = handler
}

// SessionID returns the session ID
func (t *WebSocketClientTransport) SessionID() string {
	return t.sessionID
}

// handleClose handles connection close events
func (t *WebSocketClientTransport) handleClose() {
	if t.closeHandler != nil {
		t.closeHandler()
	}
}
