package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pederhe/nca/config"
)

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the AI API
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
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

// StreamResponse represents a streaming response chunk
type StreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Client is an AI API client
type Client struct {
	apiKey      string
	apiBaseURL  string
	model       string
	temperature float64
	httpClient  *http.Client
}

// NewClient creates a new API client
func NewClient() *Client {
	apiKey := config.Get("api_key")
	apiBaseURL := config.Get("api_base_url")
	model := config.Get("model")
	temperatureStr := config.Get("temperature")

	// Set default values
	if apiBaseURL == "" {
		apiBaseURL = "https://api.deepseek.com/v1"
	}
	if model == "" {
		model = "deepseek-chat"
	}

	// 默认temperature为0
	temperature := 0.0
	if temperatureStr != "" {
		if tempValue, err := strconv.ParseFloat(temperatureStr, 64); err == nil {
			temperature = tempValue
		}
	}

	return &Client{
		apiKey:      apiKey,
		apiBaseURL:  apiBaseURL,
		model:       model,
		temperature: temperature,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for streaming
		},
	}
}

// Chat sends a conversation request to the AI API
func (c *Client) Chat(messages []Message) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("API key not set. Please use 'nca config set api_key YOUR_API_KEY' to set it")
	}

	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: c.temperature,
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

// ChatStream sends a streaming conversation request to the AI API
// It calls the callback function for each chunk of the response
func (c *Client) ChatStream(messages []Message, callback func(string, bool)) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("API key not set. Please use 'nca config set api_key YOUR_API_KEY' to set it")
	}

	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Stream:      true,
		Temperature: c.temperature,
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
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s", string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var fullContent strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fullContent.String(), err
		}

		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		content := streamResp.Choices[0].Delta.Content
		isDone := streamResp.Choices[0].FinishReason != ""

		if content != "" {
			fullContent.WriteString(content)
			callback(content, isDone)
		}

		if isDone {
			break
		}
	}

	return fullContent.String(), nil
}
