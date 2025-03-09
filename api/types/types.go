package types

import "time"

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Provider defines the interface that all AI providers must implement
type Provider interface {
	// ChatStream sends a streaming conversation request to the AI API
	// It calls the callback function for each chunk of the response
	ChatStream(messages []Message, callback func(string, string, bool)) (string, string, error)

	// GetName returns the name of the provider
	GetName() string
}

// ProviderConfig contains common configuration for providers
type ProviderConfig struct {
	APIKey      string
	APIBaseURL  string
	Model       string
	Temperature float64
	Timeout     time.Duration
	// 是否为流式请求禁用超时
	DisableStreamTimeout bool
}

// DefaultTimeout is the default timeout for API requests
const DefaultTimeout = 120 * time.Second

// StreamingTimeout is the timeout for streaming API requests
// 使用更长的超时时间用于流式请求
const StreamingTimeout = 300 * time.Second
