package common

import (
	"context"
)

// JSONRPCMessage represents a JSON-RPC message (request or response)
type JSONRPCMessage map[string]interface{}

// Transport describes the minimal contract for a MCP transport that a client or server can communicate over.
type Transport interface {
	// Start processes messages on the transport, including any connection steps that might need to be taken.
	//
	// This method should only be called after callbacks are installed, or else messages may be lost.
	//
	// NOTE: This method should not be called explicitly when using Client, Server, or Protocol classes.
	Start(ctx context.Context) error

	// Send sends a JSON-RPC message (request or response).
	Send(msg JSONRPCMessage) error

	// Close closes the connection.
	Close() error

	// SetCloseHandler sets the callback for when the connection is closed for any reason.
	// This should be invoked when Close() is called as well.
	SetCloseHandler(handler func())

	// SetErrorHandler sets the callback for when an error occurs.
	// Note that errors are not necessarily fatal; they are used for reporting any kind of exceptional condition out of band.
	SetErrorHandler(handler func(error))

	// SetMessageHandler sets the callback for when a message (request or response) is received over the connection.
	SetMessageHandler(handler func(JSONRPCMessage))

	// SessionID returns the session ID generated for this connection.
	SessionID() string
}
