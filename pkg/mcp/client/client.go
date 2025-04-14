package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pederhe/nca/pkg/mcp/common"
)

// ClientImplementation represents the client implementation information
type ClientImplementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientOptions configures the Client options
type ClientOptions struct {
	// Protocol options
	ProtocolOptions *common.ProtocolOptions

	// Client capabilities
	Capabilities map[string]interface{}
}

// Client is the MCP client implementation, based on pluggable transport
// The client will automatically start the initialization process with the server when Connect() is called
type Client struct {
	*common.Protocol
	serverCapabilities map[string]interface{}
	serverVersion      *ClientImplementation
	capabilities       map[string]interface{}
	clientInfo         ClientImplementation
	instructions       string
}

// NewClient creates a new client
func NewClient(clientInfo ClientImplementation, options *ClientOptions) *Client {
	// Handle options
	var protocolOptions *common.ProtocolOptions
	capabilities := make(map[string]interface{})

	if options != nil {
		protocolOptions = options.ProtocolOptions
		if options.Capabilities != nil {
			capabilities = options.Capabilities
		}
	}

	client := &Client{
		Protocol:     common.NewProtocol(protocolOptions),
		capabilities: capabilities,
		clientInfo:   clientInfo,
	}

	return client
}

// RegisterCapabilities registers new capabilities
// This can only be called before connecting to the transport
func (c *Client) RegisterCapabilities(capabilities map[string]interface{}) error {
	if c.Protocol.Transport() != nil {
		return errors.New("cannot register capabilities after connecting to transport")
	}

	// Merge capabilities
	merged, err := common.MergeCapabilities(c.capabilities, capabilities)
	if err != nil {
		return err
	}

	switch m := merged.(type) {
	case map[string]interface{}:
		c.capabilities = m
	default:
		return fmt.Errorf("failed to merge capabilities: expected map[string]interface{}, got %T", merged)
	}

	return nil
}

// Connect connects to the transport and performs initialization
func (c *Client) Connect(ctx context.Context, transport common.Transport) error {
	if err := c.Protocol.Connect(ctx, transport); err != nil {
		return err
	}

	// Perform initialization
	initResult, err := c.initialize()
	if err != nil {
		// If initialization fails, disconnect
		c.Protocol.Close()
		return err
	}

	c.serverCapabilities = initResult.Capabilities
	c.serverVersion = &ClientImplementation{
		Name:    initResult.ServerInfo.Name,
		Version: initResult.ServerInfo.Version,
	}
	c.instructions = initResult.Instructions

	// Send initialized notification
	return c.SendInitializedNotification()
}

// initialize sends initialization request to the server
func (c *Client) initialize() (*InitializeResult, error) {
	// Prepare initialization request
	req := common.Request{
		Method: "initialize",
		Params: map[string]interface{}{
			"protocolVersion": LatestProtocolVersion,
			"capabilities":    c.capabilities,
			"clientInfo": map[string]string{
				"name":    c.clientInfo.Name,
				"version": c.clientInfo.Version,
			},
		},
	}

	// Send request
	result, err := c.Protocol.Request(req, nil)
	if err != nil {
		return nil, err
	}

	// Validate result
	if result == nil {
		return nil, errors.New("server sent invalid initialization result: nil")
	}

	// Parse result
	var initResult InitializeResult
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	if err := json.Unmarshal(resultJSON, &initResult); err != nil {
		return nil, fmt.Errorf("failed to parse initialization result: %w", err)
	}

	// Check protocol version
	if !isSupportedProtocolVersion(initResult.ProtocolVersion) {
		return nil, fmt.Errorf("server protocol version not supported: %s", initResult.ProtocolVersion)
	}

	return &initResult, nil
}

// SendInitializedNotification sends the initialized notification
func (c *Client) SendInitializedNotification() error {
	notification := common.Notification{
		Method: "notifications/initialized",
	}
	return c.Protocol.Notification(notification)
}

// SendInitializedNotificationDirect directly sends the initialized notification for testing
// It bypasses Protocol.Notification method and directly uses transport.Send
func (c *Client) SendInitializedNotificationDirect() error {
	if c.Protocol.Transport() == nil {
		return errors.New("not connected")
	}

	message := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}

	return c.Protocol.Transport().Send(message)
}

// GetServerCapabilities returns the capabilities reported by the server
func (c *Client) GetServerCapabilities() map[string]interface{} {
	return c.serverCapabilities
}

// GetServerVersion returns the server version information
func (c *Client) GetServerVersion() *ClientImplementation {
	return c.serverVersion
}

// GetInstructions returns the server instructions
func (c *Client) GetInstructions() string {
	return c.instructions
}

// assertCapability checks if the server supports the specified capability
func (c *Client) assertCapability(capability string, method string) error {
	if c.serverCapabilities == nil {
		return errors.New("client not initialized")
	}

	_, exists := c.serverCapabilities[capability]
	if !exists {
		return fmt.Errorf("server does not support %s (required for %s)", capability, method)
	}

	return nil
}

// Ping sends a ping request
func (c *Client) Ping(ctx context.Context) error {
	req := common.Request{
		Method: "ping",
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	_, err := c.Protocol.Request(req, options)
	return err
}

// Complete sends a complete request
func (c *Client) Complete(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if err := c.assertCapability("completion", "complete"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "complete",
		Params: params,
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// SetLoggingLevel sets the logging level
func (c *Client) SetLoggingLevel(ctx context.Context, level string) error {
	if err := c.assertCapability("logging", "logging/setLevel"); err != nil {
		return err
	}

	req := common.Request{
		Method: "logging/setLevel",
		Params: map[string]interface{}{
			"level": level,
		},
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	_, err := c.Protocol.Request(req, options)
	return err
}

// GetPrompt gets the specified prompt
func (c *Client) GetPrompt(ctx context.Context, id string) (map[string]interface{}, error) {
	if err := c.assertCapability("prompts", "prompts/get"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "prompts/get",
		Params: map[string]interface{}{
			"id": id,
		},
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// ListPrompts lists all available prompts
func (c *Client) ListPrompts(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if err := c.assertCapability("prompts", "prompts/list"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "prompts/list",
		Params: params,
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// ListResources lists all available resources
func (c *Client) ListResources(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if err := c.assertCapability("resources", "resources/list"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "resources/list",
		Params: params,
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// ListResourceTemplates lists all available resource templates
func (c *Client) ListResourceTemplates(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if err := c.assertCapability("resources", "resources/templates/list"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "resources/templates/list",
		Params: params,
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// ReadResource reads the specified resource
func (c *Client) ReadResource(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if err := c.assertCapability("resources", "resources/read"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "resources/read",
		Params: params,
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// SubscribeResource subscribes to the specified resource
func (c *Client) SubscribeResource(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if err := c.assertCapability("resources", "resources/subscribe"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "resources/subscribe",
		Params: params,
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// UnsubscribeResource unsubscribes from the specified resource
func (c *Client) UnsubscribeResource(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if err := c.assertCapability("resources", "resources/unsubscribe"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "resources/unsubscribe",
		Params: params,
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// CallTool calls a tool
func (c *Client) CallTool(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if err := c.assertCapability("tools", "tools/call"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "tools/call",
		Params: params,
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// ListTools lists all available tools
func (c *Client) ListTools(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if err := c.assertCapability("tools", "tools/list"); err != nil {
		return nil, err
	}

	req := common.Request{
		Method: "tools/list",
		Params: params,
	}

	options := &common.RequestOptions{
		Signal: ctx,
	}

	return c.Protocol.Request(req, options)
}

// SendRootsListChanged sends the roots list changed notification
func (c *Client) SendRootsListChanged() error {
	notification := common.Notification{
		Method: "notifications/roots/list_changed",
	}
	return c.Protocol.Notification(notification)
}

// InitializeResult is the result of the initialization request
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	ServerInfo      ClientImplementation   `json:"serverInfo"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	Instructions    string                 `json:"instructions,omitempty"`
}

// Protocol version constants
const (
	LatestProtocolVersion = "2024-11-05"
)

// Supported protocol versions
var supportedProtocolVersions = []string{
	"2024-11-05",
}

// isSupportedProtocolVersion checks if the given protocol version is supported
func isSupportedProtocolVersion(version string) bool {
	for _, v := range supportedProtocolVersions {
		if v == version {
			return true
		}
	}
	return false
}
