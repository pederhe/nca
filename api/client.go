package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/nca/config"
)

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the AI API
type ChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

// ChatResponse represents a response from the AI API
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Client is an AI API client
type Client struct {
	apiKey     string
	apiBaseURL string
	model      string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient() *Client {
	apiKey := config.Get("api_key")
	apiBaseURL := config.Get("api_base_url")
	model := config.Get("model")

	// Set default values
	if apiBaseURL == "" {
		apiBaseURL = "https://api.deepseek.com/v1"
	}
	if model == "" {
		model = "deepseek-chat"
	}

	return &Client{
		apiKey:     apiKey,
		apiBaseURL: apiBaseURL,
		model:      model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Chat sends a conversation request to the AI API
func (c *Client) Chat(messages []Message) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("API key not set. Please use 'nca config set api_key YOUR_API_KEY' to set it")
	}

	reqBody := ChatRequest{
		Model:    c.model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.apiBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API returned empty response")
	}

	return chatResp.Choices[0].Message.Content, nil
}
