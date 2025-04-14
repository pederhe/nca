package common

// McpServer represents an MCP server configuration and status
type McpServer struct {
	Name              string                `json:"name"`
	Config            string                `json:"config"`
	Status            string                `json:"status"` // "connected", "connecting", "disconnected"
	Error             string                `json:"error,omitempty"`
	Tools             []McpTool             `json:"tools,omitempty"`
	Resources         []McpResource         `json:"resources,omitempty"`
	ResourceTemplates []McpResourceTemplate `json:"resourceTemplates,omitempty"`
	Disabled          bool                  `json:"disabled,omitempty"`
	Timeout           int                   `json:"timeout,omitempty"`
}

// McpTool represents a tool provided by an MCP server
type McpTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
	AutoApprove bool        `json:"autoApprove,omitempty"`
}

// McpResource represents a resource provided by an MCP server
type McpResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	MimeType    string `json:"mimeType,omitempty"`
	Description string `json:"description,omitempty"`
}

// McpResourceTemplate represents a resource template for an MCP server
type McpResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// McpResourceResponse represents a response containing resource content
type McpResourceResponse struct {
	Meta     map[string]interface{} `json:"_meta,omitempty"`
	Contents []ResourceContent      `json:"contents"`
}

// ResourceContent represents the content of a resource
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// McpToolCallResponse represents a response from a tool call
type McpToolCallResponse struct {
	Meta    map[string]interface{} `json:"_meta,omitempty"`
	Content []ToolResponseContent  `json:"content"`
	IsError bool                   `json:"isError,omitempty"`
}

// ToolResponseContent represents different types of content in a tool response
type ToolResponseContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Data     string          `json:"data,omitempty"`
	MimeType string          `json:"mimeType,omitempty"`
	Resource ResourceContent `json:"resource,omitempty"`
}
