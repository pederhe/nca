package hub

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Constants definition
const (
	DEFAULT_MCP_TIMEOUT_SECONDS = 60
	MIN_MCP_TIMEOUT_SECONDS     = 10
)

// ServerConfig represents the configuration of an MCP server
type ServerConfig struct {
	TransportType McpTransportType `json:"transportType"`
	AutoApprove   []string         `json:"autoApprove,omitempty"`
	Disabled      bool             `json:"disabled,omitempty"`
	Timeout       int              `json:"timeout,omitempty"`

	// Stdio specific configuration
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	// SSE specific configuration
	URL string `json:"url,omitempty"`
}

// Validate checks if the configuration is valid
func (c *ServerConfig) Validate() error {
	// Validate timeout settings
	if c.Timeout == 0 {
		c.Timeout = DEFAULT_MCP_TIMEOUT_SECONDS
	} else if c.Timeout < MIN_MCP_TIMEOUT_SECONDS {
		return fmt.Errorf("timeout must be at least %d seconds", MIN_MCP_TIMEOUT_SECONDS)
	}

	// Validate transport type specific configuration
	switch c.TransportType {
	case TransportTypeStdio:
		if c.Command == "" {
			return errors.New("command is required for stdio transport")
		}
	case TransportTypeSSE:
		if c.URL == "" {
			return errors.New("url is required for sse transport")
		}
	default:
		return fmt.Errorf("unsupported transport type: %s", c.TransportType)
	}

	return nil
}

// McpSettings represents the structure of MCP settings file
type McpSettings struct {
	McpServers map[string]*ServerConfig `json:"mcp_servers"`
}

// ParseSettings parses the content of MCP settings file
func ParseSettings(content []byte) (*McpSettings, error) {
	var settings McpSettings
	if err := json.Unmarshal(content, &settings); err != nil {
		return nil, fmt.Errorf("invalid MCP settings format: %w", err)
	}

	if settings.McpServers == nil {
		settings.McpServers = make(map[string]*ServerConfig)
	}

	// Validate all server configurations
	for name, config := range settings.McpServers {
		if err := config.Validate(); err != nil {
			return nil, fmt.Errorf("invalid configuration for server '%s': %w", name, err)
		}
	}

	return &settings, nil
}
