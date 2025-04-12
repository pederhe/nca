package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pederhe/nca/api/providers"
	"github.com/pederhe/nca/api/types"
	"github.com/pederhe/nca/config"
)

// ProviderType represents the type of AI provider
type ProviderType string

const (
	// DeepSeekProvider is the DeepSeek AI provider
	DeepSeekProvider ProviderType = "deepseek"
	// QwenProvider is the Qwen AI provider
	QwenProvider ProviderType = "qwen"
	// DouBaoProvider is the DouBao AI provider
	DouBaoProvider ProviderType = "doubao"
)

// GetProvider returns a provider based on the provider type
func GetProvider(providerType ProviderType) (types.Provider, error) {
	apiKey := config.Get("api_key")
	apiBaseURL := config.Get("api_base_url")
	model := config.Get("model")
	temperatureStr := config.Get("temperature")

	temperature := 0.0
	if temperatureStr != "" {
		if tempValue, err := strconv.ParseFloat(temperatureStr, 64); err == nil {
			temperature = tempValue
		}
	}

	// Read configuration for disabling stream timeout
	disableStreamTimeoutStr := config.Get("disable_stream_timeout")
	disableStreamTimeout := false
	if disableStreamTimeoutStr == "true" || disableStreamTimeoutStr == "1" {
		disableStreamTimeout = true
	}

	providerConfig := types.ProviderConfig{
		APIKey:               apiKey,
		APIBaseURL:           apiBaseURL,
		Model:                model,
		Temperature:          temperature,
		Timeout:              types.DefaultTimeout,
		DisableStreamTimeout: disableStreamTimeout,
	}

	switch providerType {
	case DeepSeekProvider:
		return providers.NewDeepSeekProvider(providerConfig)
	case QwenProvider:
		return providers.NewQwenProvider(providerConfig)
	case DouBaoProvider:
		return providers.NewDouBaoProvider(providerConfig)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetDefaultProvider returns the default provider based on configuration
func GetDefaultProvider() (types.Provider, error) {
	providerName := config.Get("provider")
	if providerName != "" {
		return GetProvider(ProviderType(providerName))
	}

	// Determine provider based on model name keywords
	model := config.Get("model")
	if model != "" {
		if strings.Contains(strings.ToLower(model), "deepseek") {
			providerName = string(DeepSeekProvider)
		} else if strings.Contains(strings.ToLower(model), "qwen") {
			providerName = string(QwenProvider)
		} else if strings.Contains(strings.ToLower(model), "doubao") {
			providerName = string(DouBaoProvider)
		}
		// Additional model matching logic can be added here
	}

	if providerName == "" {
		providerName = string(DeepSeekProvider) // Default to DeepSeek
	}

	return GetProvider(ProviderType(providerName))
}
