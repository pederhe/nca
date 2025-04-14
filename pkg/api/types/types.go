package types

import (
	"context"
	"time"
)

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatStreamResponse represents the response from a streaming chat request
type ChatStreamResponse struct {
	ReasoningContent string `json:"reasoning_content"`
	Content          string `json:"content"`
	Usage            *Usage `json:"usage,omitempty"`
	FinishReason     string `json:"finish_reason,omitempty"`
}

// Provider defines the interface that all AI providers must implement
type Provider interface {
	// ChatStream sends a streaming conversation request to the AI API
	// It calls the callback function for each chunk of the response
	// The context parameter allows for cancellation of the request
	ChatStream(ctx context.Context, messages []Message, callback func(string, string, bool)) (*ChatStreamResponse, error)

	// GetName returns the name of the provider
	GetName() string

	// GetModelInfo returns information about the model
	GetModelInfo() *ModelInfo
}

// ProviderConfig contains common configuration for providers
type ProviderConfig struct {
	APIKey      string
	APIBaseURL  string
	Model       string
	Temperature float64
	Timeout     time.Duration
	// Whether to disable timeout for streaming requests
	DisableStreamTimeout bool
}

// DefaultTimeout is the default timeout for API requests
const DefaultTimeout = 120 * time.Second

// StreamingTimeout is the timeout for streaming API requests
// Use a longer timeout for streaming requests
const StreamingTimeout = 300 * time.Second
