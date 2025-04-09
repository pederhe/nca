package hub

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pederhe/nca/config"
	"github.com/pederhe/nca/core/mcp/client"
	"github.com/pederhe/nca/core/mcp/common"
)

// McpTransportType defines the MCP transport type
type McpTransportType string

const (
	TransportTypeStdio McpTransportType = "stdio"
	TransportTypeSSE   McpTransportType = "sse"
)

// Default timeout for internal MCP data requests (in milliseconds)
const DEFAULT_REQUEST_TIMEOUT_MS = 5000

// McpConnection represents a connection to an MCP server
type McpConnection struct {
	Server    common.McpServer
	Client    *client.Client
	Transport interface{} // Can be either StdioClientTransport or SSEClientTransport
}

// McpHub manages multiple MCP server connections
type McpHub struct {
	connections []*McpConnection
}

// McpHub instance
var hub *McpHub

func GetMcpHub() *McpHub {
	if hub == nil {
		hub = &McpHub{
			connections: make([]*McpConnection, 0),
		}
		go hub.initializeMcpServers()
	}

	return hub
}

// GetServers returns all enabled servers
func (h *McpHub) GetServers() []common.McpServer {
	servers := make([]common.McpServer, 0)
	for _, conn := range h.connections {
		if !conn.Server.Disabled {
			servers = append(servers, conn.Server)
		}
	}
	return servers
}

// GetMode returns the MCP mode
func (h *McpHub) GetMode() string {
	mcp_mode := config.Get("mcp_mode")
	if mcp_mode == "" {
		return "off"
	}
	return mcp_mode
}

// getMcpSettingsFilePath gets the path to the MCP settings file
func (h *McpHub) getMcpSettingsFilePath() string {
	path := config.Get("mcp_settings_file")
	if path == "" {
		path = filepath.Join(os.Getenv("HOME"), ".nca", "mcp_settings.json")
	}
	return path
}

// initializeMcpServers initializes MCP server connections
func (h *McpHub) initializeMcpServers() {
	settingsPath := h.getMcpSettingsFilePath()

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		fmt.Printf("Error reading settings file: %v\n", err)
		return
	}

	settings, err := ParseSettings(content)
	if err != nil {
		fmt.Printf("Error parsing settings: %v\n", err)
		return
	}

	if err := h.updateServerConnections(settings.McpServers); err != nil {
		fmt.Printf("Error initializing server connections: %v\n", err)
	}

	//h.printConnections()
}

// Dispose closes all resources
func (h *McpHub) Dispose() error {
	// Close all connections
	for _, conn := range h.connections {
		if conn.Client != nil {
			// close client connection
			conn.Client.Close()

			// for stdio transport, explicitly check and terminate subprocess
			if stdioTransport, ok := conn.Transport.(*client.StdioClientTransport); ok {
				// call Close method - already modified to ensure subprocess is terminated
				stdioTransport.Close()
			}
		}
	}
	h.connections = make([]*McpConnection, 0)

	return nil
}

// printConnections prints current MCP server connections in a formatted way
func (h *McpHub) printConnections() {
	fmt.Println("\nCurrent MCP Server Connections:")
	fmt.Println("--------------------------------")
	for _, conn := range h.connections {
		fmt.Printf("Server: %s\n", conn.Server.Name)
		fmt.Printf("  Status: %s\n", conn.Server.Status)
		if conn.Server.Error != "" {
			fmt.Printf("  Error: %s\n", conn.Server.Error)
		}
		fmt.Printf("  Transport: %s\n", conn.Server.Config)

		// Print tools information
		if len(conn.Server.Tools) > 0 {
			fmt.Println("  Tools:")
			for _, tool := range conn.Server.Tools {
				fmt.Printf("    - %s: %s\n", tool.Name, tool.Description)
			}
		}

		fmt.Println("--------------------------------")
	}
}
