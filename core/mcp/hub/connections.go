package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/pederhe/nca/core/mcp/client"
	"github.com/pederhe/nca/core/mcp/common"
)

// connectToServer connects to an MCP server
func (h *McpHub) connectToServer(name string, config *ServerConfig) error {
	// First remove any existing connection with the same name
	for i, conn := range h.connections {
		if conn.Server.Name == name {
			if conn.Client != nil {
				conn.Client.Close()
			}
			h.connections = append(h.connections[:i], h.connections[i+1:]...)
			break
		}
	}

	// Create new client
	clientInfo := client.ClientImplementation{
		Name:    "NCA",
		Version: "1.0.0",
	}

	clientOptions := &client.ClientOptions{
		Capabilities: make(map[string]interface{}),
	}

	mcpClient := client.NewClient(clientInfo, clientOptions)

	// Create connection
	connection := &McpConnection{
		Server: common.McpServer{
			Name:     name,
			Config:   string(mustMarshalJSON(config)),
			Status:   "connecting",
			Disabled: config.Disabled,
			Timeout:  config.Timeout,
		},
		Client: mcpClient,
	}

	// Add to connections list
	h.connections = append(h.connections, connection)

	// Create transport object based on different transport types
	var transport common.Transport
	var err error

	ctx := context.Background()

	switch config.TransportType {
	case TransportTypeStdio:
		// Create transport for stdio
		params := client.StdioServerParameters{
			Command: config.Command,
			Args:    config.Args,
			Env:     config.Env,
		}

		// Use default environment if no environment variables are set
		if params.Env == nil {
			params.Env = client.GetDefaultEnvironment()
		}

		stdioTransport := client.NewStdioClientTransport(params)

		// Set error handler
		stdioTransport.SetErrorHandler(func(err error) {
			h.appendErrorMessage(connection, err.Error())
			connection.Server.Status = "disconnected"
		})

		// Set close handler
		stdioTransport.SetCloseHandler(func() {
			connection.Server.Status = "disconnected"
		})

		transport = stdioTransport
		connection.Transport = stdioTransport

	case TransportTypeSSE:
		// Create transport for SSE
		sseURL, err := url.Parse(config.URL)
		if err != nil {
			connection.Server.Status = "disconnected"
			h.appendErrorMessage(connection, fmt.Sprintf("invalid URL: %v", err))
			return err
		}

		sseOptions := &client.SSEClientTransportOptions{}
		sseTransport := client.NewSSEClientTransport(sseURL, sseOptions)

		// Set error handler
		sseTransport.SetErrorHandler(func(err error) {
			h.appendErrorMessage(connection, err.Error())
			connection.Server.Status = "disconnected"
		})

		// Set close handler
		sseTransport.SetCloseHandler(func() {
			connection.Server.Status = "disconnected"
		})

		transport = sseTransport
		connection.Transport = sseTransport

	default:
		connection.Server.Status = "disconnected"
		errMsg := fmt.Sprintf("unsupported transport type: %s", config.TransportType)
		h.appendErrorMessage(connection, errMsg)
		return errors.New(errMsg)
	}

	// Attempt to connect
	if err := mcpClient.Connect(ctx, transport); err != nil {
		connection.Server.Status = "disconnected"
		h.appendErrorMessage(connection, err.Error())
		return err
	}

	// Set status after successful connection
	connection.Server.Status = "connected"
	connection.Server.Error = ""

	// Initially fetch tools and resources lists
	tools, err := h.fetchToolsList(name)
	if err != nil {
		h.appendErrorMessage(connection, fmt.Sprintf("Failed to fetch tools: %v", err))
	} else {
		connection.Server.Tools = tools
	}

	resources, err := h.fetchResourcesList(name)
	if err != nil {
		h.appendErrorMessage(connection, fmt.Sprintf("Failed to fetch resources: %v", err))
	} else {
		connection.Server.Resources = resources
	}

	templates, err := h.fetchResourceTemplatesList(name)
	if err != nil {
		h.appendErrorMessage(connection, fmt.Sprintf("Failed to fetch resource templates: %v", err))
	} else {
		connection.Server.ResourceTemplates = templates
	}

	return nil
}

// appendErrorMessage appends error message to server's error messages
func (h *McpHub) appendErrorMessage(connection *McpConnection, errMsg string) {
	if connection.Server.Error == "" {
		connection.Server.Error = errMsg
	} else {
		connection.Server.Error = connection.Server.Error + "\n" + errMsg
	}
}

// updateServerConnections updates server connections
func (h *McpHub) updateServerConnections(newServers map[string]*ServerConfig) error {
	// Get current and new server names
	currentNames := make(map[string]bool)
	for _, conn := range h.connections {
		currentNames[conn.Server.Name] = true
	}

	// Remove deleted servers
	for name := range currentNames {
		if _, exists := newServers[name]; !exists {
			if err := h.deleteConnection(name); err != nil {
				fmt.Printf("Error deleting connection %s: %v\n", name, err)
			}
			fmt.Printf("Deleted MCP server: %s\n", name)
		}
	}

	// Update or add servers
	for name, config := range newServers {
		// Check if existing connection needs update
		needsUpdate := true
		for _, conn := range h.connections {
			if conn.Server.Name == name {
				// Compare if configuration has changed
				var currentConfig ServerConfig
				if err := json.Unmarshal([]byte(conn.Server.Config), &currentConfig); err != nil {
					fmt.Printf("Error unmarshaling current config for %s: %v\n", name, err)
					continue
				}

				// If configuration is the same, no update needed
				if configEquals(&currentConfig, config) {
					needsUpdate = false
					break
				}
			}
		}

		if needsUpdate {
			// Delete old connection if exists
			if err := h.deleteConnection(name); err != nil {
				fmt.Printf("Error deleting connection before update %s: %v\n", name, err)
			}

			// Create new connection
			if err := h.connectToServer(name, config); err != nil {
				fmt.Printf("Failed to connect to MCP server %s: %v\n", name, err)
			}
		}
	}

	return nil
}

// RestartConnection restarts the connection for a specified server
func (h *McpHub) RestartConnection(serverName string) error {
	// Find existing connection
	var connection *McpConnection
	for _, conn := range h.connections {
		if conn.Server.Name == serverName {
			connection = conn
			break
		}
	}

	if connection == nil {
		return fmt.Errorf("no connection found for server: %s", serverName)
	}

	// Parse configuration
	var config ServerConfig
	if err := json.Unmarshal([]byte(connection.Server.Config), &config); err != nil {
		return fmt.Errorf("failed to parse server config: %w", err)
	}

	// Update connection status
	connection.Server.Status = "connecting"
	connection.Server.Error = ""

	// Add a small delay so user can see the restart process
	time.Sleep(500 * time.Millisecond)

	// Delete old connection
	if err := h.deleteConnection(serverName); err != nil {
		fmt.Printf("Error deleting connection during restart: %v\n", err)
	}

	// Create new connection
	if err := h.connectToServer(serverName, &config); err != nil {
		fmt.Printf("Failed to reconnect to server %s: %v\n", serverName, err)
		return err
	}

	return nil
}

// deleteConnection deletes connection with specified name
func (h *McpHub) deleteConnection(name string) error {
	for i, conn := range h.connections {
		if conn.Server.Name == name {
			if conn.Client != nil {
				conn.Client.Close()
			}
			h.connections = append(h.connections[:i], h.connections[i+1:]...)
			return nil
		}
	}

	return nil
}

// Helper function - must be able to convert object to JSON
func mustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal json: %v", err))
	}
	return data
}

// configEquals compares if two configurations are equal
func configEquals(a, b *ServerConfig) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}

	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}

	return string(aJSON) == string(bJSON)
}
