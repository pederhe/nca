package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pederhe/nca/api/types"
)

// DeepSeekProvider implements the Provider interface for DeepSeek AI
type DeepSeekProvider struct {
	apiKey               string
	apiBaseURL           string
	model                string
	temperature          float64
	disableStreamTimeout bool
}

// ChatRequest represents a request to the DeepSeek API
type deepSeekChatRequest struct {
	Model       string          `json:"model"`
	Messages    []types.Message `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

// StreamResponse represents a streaming response chunk from DeepSeek
type deepSeekStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Delta struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// NewDeepSeekProvider creates a new DeepSeek provider
func NewDeepSeekProvider(config types.ProviderConfig) *DeepSeekProvider {
	// Set default values if not provided
	baseURL := config.APIBaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}

	model := config.Model
	if model == "" {
		model = "deepseek-chat"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = types.DefaultTimeout
	}

	return &DeepSeekProvider{
		apiKey:               config.APIKey,
		apiBaseURL:           baseURL,
		model:                model,
		temperature:          config.Temperature,
		disableStreamTimeout: config.DisableStreamTimeout,
	}
}

// GetName returns the name of the provider
func (p *DeepSeekProvider) GetName() string {
	return "deepseek"
}

// ChatStream sends a streaming conversation request to the DeepSeek API
func (p *DeepSeekProvider) ChatStream(ctx context.Context, messages []types.Message, callback func(string, string, bool)) (string, string, error) {
	if p.apiKey == "" {
		return "", "", fmt.Errorf("API key not set for DeepSeek provider")
	}

	reqBody := deepSeekChatRequest{
		Model:       p.model,
		Messages:    messages,
		Stream:      true,
		Temperature: p.temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", err
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	// Create an HTTP client for streaming requests
	var streamClient *http.Client

	if p.disableStreamTimeout {
		// HTTP client without timeout
		streamClient = &http.Client{
			Timeout: 0, // 0 means no timeout
		}
	} else {
		// Use a longer timeout for streaming
		streamClient = &http.Client{
			Timeout: types.StreamingTimeout,
		}
	}

	resp, err := streamClient.Do(req)
	if err != nil {
		// Check if the error is due to context cancellation
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("DeepSeek API error: %s", string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var fullContent strings.Builder
	var fullReasoningContent strings.Builder

	// Create a channel for handling context cancellation
	done := make(chan struct{})
	defer close(done)

	// Monitor context cancellation in a goroutine
	go func() {
		select {
		case <-ctx.Done():
			// Context was cancelled, close the response body
			resp.Body.Close()
		case <-done:
			// Normal completion, do nothing
		}
	}()

	for {
		// Check if context has been cancelled
		select {
		case <-ctx.Done():
			return fullReasoningContent.String(), fullContent.String(), ctx.Err()
		default:
			// Continue processing
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			// Check if the error is due to context cancellation
			if ctx.Err() != nil {
				return fullReasoningContent.String(), fullContent.String(), ctx.Err()
			}
			return fullReasoningContent.String(), fullContent.String(), err
		}

		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var streamResp deepSeekStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		reasoningContent := streamResp.Choices[0].Delta.ReasoningContent
		content := streamResp.Choices[0].Delta.Content
		isDone := streamResp.Choices[0].FinishReason != ""

		if reasoningContent != "" {
			fullReasoningContent.WriteString(reasoningContent)
		}

		if content != "" {
			fullContent.WriteString(content)
		}

		callback(reasoningContent, content, isDone)

		if isDone {
			break
		}
	}

	return fullReasoningContent.String(), fullContent.String(), nil
}
