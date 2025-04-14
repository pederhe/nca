package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pederhe/nca/pkg/mcp/common"
)

// fetchToolsList retrieves the list of tools provided by the server
func (h *McpHub) fetchToolsList(serverName string) ([]common.McpTool, error) {
	var connection *McpConnection
	for _, conn := range h.connections {
		if conn.Server.Name == serverName {
			connection = conn
			break
		}
	}

	if connection == nil {
		return nil, fmt.Errorf("no connection found for server: %s", serverName)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(DEFAULT_REQUEST_TIMEOUT_MS)*time.Millisecond)
	defer cancel()

	// Call the ListTools method
	response, err := connection.Client.ListTools(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result struct {
		Tools []struct {
			Name        string      `json:"name"`
			Description string      `json:"description,omitempty"`
			InputSchema interface{} `json:"inputSchema,omitempty"`
		} `json:"tools"`
	}

	resultJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools response: %w", err)
	}

	settingsPath := h.getMcpSettingsFilePath()

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings McpSettings
	if err := json.Unmarshal(content, &settings); err != nil {
		return nil, err
	}

	autoApproveConfig := make(map[string]bool)
	if serverConfig, ok := settings.McpServers[serverName]; ok && serverConfig.AutoApprove != nil {
		for _, toolName := range serverConfig.AutoApprove {
			autoApproveConfig[toolName] = true
		}
	}

	// Build the tools list, marking auto-approved tools
	tools := make([]common.McpTool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		tools = append(tools, common.McpTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
			AutoApprove: autoApproveConfig[tool.Name],
		})
	}

	return tools, nil
}

// fetchResourcesList retrieves the list of resources provided by the server
func (h *McpHub) fetchResourcesList(serverName string) ([]common.McpResource, error) {
	var connection *McpConnection
	for _, conn := range h.connections {
		if conn.Server.Name == serverName {
			connection = conn
			break
		}
	}

	if connection == nil {
		return nil, fmt.Errorf("no connection found for server: %s", serverName)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(DEFAULT_REQUEST_TIMEOUT_MS)*time.Millisecond)
	defer cancel()

	// Call the ListResources method
	response, err := connection.Client.ListResources(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result struct {
		Resources []struct {
			URI         string `json:"uri"`
			Name        string `json:"name"`
			MimeType    string `json:"mimeType,omitempty"`
			Description string `json:"description,omitempty"`
		} `json:"resources"`
	}

	resultJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resources response: %w", err)
	}

	// Build the resources list
	resources := make([]common.McpResource, 0, len(result.Resources))
	for _, res := range result.Resources {
		resources = append(resources, common.McpResource{
			URI:         res.URI,
			Name:        res.Name,
			MimeType:    res.MimeType,
			Description: res.Description,
		})
	}

	return resources, nil
}

// fetchResourceTemplatesList retrieves the list of resource templates provided by the server
func (h *McpHub) fetchResourceTemplatesList(serverName string) ([]common.McpResourceTemplate, error) {
	var connection *McpConnection
	for _, conn := range h.connections {
		if conn.Server.Name == serverName {
			connection = conn
			break
		}
	}

	if connection == nil {
		return nil, fmt.Errorf("no connection found for server: %s", serverName)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(DEFAULT_REQUEST_TIMEOUT_MS)*time.Millisecond)
	defer cancel()

	// Call the ListResourceTemplates method
	response, err := connection.Client.ListResourceTemplates(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result struct {
		ResourceTemplates []struct {
			URITemplate string `json:"uriTemplate"`
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			MimeType    string `json:"mimeType,omitempty"`
		} `json:"resourceTemplates"`
	}

	resultJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resource templates response: %w", err)
	}

	// Build the resource templates list
	templates := make([]common.McpResourceTemplate, 0, len(result.ResourceTemplates))
	for _, tmpl := range result.ResourceTemplates {
		templates = append(templates, common.McpResourceTemplate{
			URITemplate: tmpl.URITemplate,
			Name:        tmpl.Name,
			Description: tmpl.Description,
			MimeType:    tmpl.MimeType,
		})
	}

	return templates, nil
}

// ReadResource reads the content of a resource
func (h *McpHub) ReadResource(serverName string, uri string) (*common.McpResourceResponse, error) {
	var connection *McpConnection
	for _, conn := range h.connections {
		if conn.Server.Name == serverName {
			connection = conn
			break
		}
	}

	if connection == nil {
		return nil, fmt.Errorf("no connection found for server: %s", serverName)
	}

	if connection.Server.Disabled {
		return nil, fmt.Errorf("server \"%s\" is disabled", serverName)
	}

	// Create context
	ctx := context.Background()

	// Call the ReadResource method
	response, err := connection.Client.ReadResource(ctx, map[string]interface{}{
		"uri": uri,
	})
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result common.McpResourceResponse

	resultJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resource response: %w", err)
	}

	return &result, nil
}

// CallTool invokes a tool
func (h *McpHub) CallTool(serverName string, toolName string, toolArguments map[string]interface{}) (*common.McpToolCallResponse, error) {
	var connection *McpConnection
	for _, conn := range h.connections {
		if conn.Server.Name == serverName {
			connection = conn
			break
		}
	}

	if connection == nil {
		return nil, fmt.Errorf("no connection found for server: %s. Please make sure to use MCP servers available under 'Connected MCP Servers'", serverName)
	}

	if connection.Server.Disabled {
		return nil, fmt.Errorf("server \"%s\" is disabled and cannot be used", serverName)
	}

	// Get timeout settings
	timeout := DEFAULT_MCP_TIMEOUT_SECONDS * 1000 // Convert to milliseconds

	if serverConfig, ok := connection.Server.Timeout, true; ok {
		timeout = serverConfig * 1000
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// Call the CallTool method
	response, err := connection.Client.CallTool(ctx, map[string]interface{}{
		"name":      toolName,
		"arguments": toolArguments,
	})
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result common.McpToolCallResponse

	resultJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool call response: %w", err)
	}

	return &result, nil
}
