package api

import (
	"context"

	"github.com/pederhe/nca/pkg/api/types"
)

// Client is a wrapper around the AI provider
type Client struct {
	provider types.Provider
}

// NewClient creates a new API client with the default provider
func NewClient() (*Client, error) {
	provider, err := GetDefaultProvider()
	if err != nil {
		return nil, err
	}

	return &Client{
		provider: provider,
	}, nil
}

// NewClientWithProvider creates a new API client with a specific provider
func NewClientWithProvider(providerType ProviderType) (*Client, error) {
	provider, err := GetProvider(providerType)
	if err != nil {
		return nil, err
	}

	return &Client{
		provider: provider,
	}, nil
}

// ChatStream sends a streaming conversation request to the AI API
func (c *Client) ChatStream(ctx context.Context, messages []types.Message, callback func(string, string, bool)) (*types.ChatStreamResponse, error) {
	return c.provider.ChatStream(ctx, messages, callback)
}

func (c *Client) GetModelInfo() *types.ModelInfo {
	return c.provider.GetModelInfo()
}

func (c *Client) GetName() string {
	return c.provider.GetName()
}
