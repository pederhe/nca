package types

// PriceTier represents a pricing tier for input/output tokens
type PriceTier struct {
	MaxTokens int     `json:"maxTokens"`
	Price     float64 `json:"price"`
}

// ModelInfo represents information about an AI model
type ModelInfo struct {
	Name                string      `json:"name"`
	MaxTokens           *int        `json:"maxTokens,omitempty"`
	ContextWindow       *int        `json:"contextWindow,omitempty"`
	SupportsImages      *bool       `json:"supportsImages,omitempty"`
	SupportsComputerUse *bool       `json:"supportsComputerUse,omitempty"`
	SupportsPromptCache bool        `json:"supportsPromptCache"`
	InputPrice          *float64    `json:"inputPrice,omitempty"`
	InputPriceTiers     []PriceTier `json:"inputPriceTiers,omitempty"`
	OutputPrice         *float64    `json:"outputPrice,omitempty"`
	OutputPriceTiers    []PriceTier `json:"outputPriceTiers,omitempty"`
	CacheWritesPrice    *float64    `json:"cacheWritesPrice,omitempty"`
	CacheReadsPrice     *float64    `json:"cacheReadsPrice,omitempty"`
	Description         *string     `json:"description,omitempty"`
}

// DeepSeekModelID represents the type of DeepSeek model IDs
type DeepSeekModelID string

const (
	// DeepSeekDefaultModelID is the default model ID for DeepSeek
	DeepSeekDefaultModelID DeepSeekModelID = "deepseek-chat"
)

// DeepSeekModels contains information about all available DeepSeek models
var DeepSeekModels = map[DeepSeekModelID]ModelInfo{
	"deepseek-chat": {
		MaxTokens:           ptr(8000),
		ContextWindow:       ptr(64000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: true,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(1.1),
		CacheWritesPrice:    ptr(0.27),
		CacheReadsPrice:     ptr(0.07),
	},
	"deepseek-reasoner": {
		MaxTokens:           ptr(8000),
		ContextWindow:       ptr(64000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: true,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(2.19),
		CacheWritesPrice:    ptr(0.55),
		CacheReadsPrice:     ptr(0.14),
	},
}

// DoubaoModelID represents the type of Doubao model IDs
type DoubaoModelID string

const (
	// DoubaoDefaultModelID is the default model ID for Doubao
	DoubaoDefaultModelID DoubaoModelID = "doubao-1-5-pro-256k-250115"
)

// DoubaoModels contains information about all available Doubao models
var DoubaoModels = map[DoubaoModelID]ModelInfo{
	"doubao-1-5-pro-256k-250115": {
		MaxTokens:           ptr(12288),
		ContextWindow:       ptr(256000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.7),
		OutputPrice:         ptr(1.3),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
	"doubao-1-5-pro-32k-250115": {
		MaxTokens:           ptr(12288),
		ContextWindow:       ptr(32000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.11),
		OutputPrice:         ptr(0.3),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
}

// QwenModelID represents the type of Qwen model IDs
type QwenModelID string

const (
	// InternationalQwenDefaultModelID is the default model ID for international Qwen
	InternationalQwenDefaultModelID QwenModelID = "qwen-coder-plus-latest"
	// MainlandQwenDefaultModelID is the default model ID for mainland Qwen
	MainlandQwenDefaultModelID QwenModelID = "qwen-coder-plus-latest"
)

// InternationalQwenModels contains information about all available international Qwen models
var InternationalQwenModels = map[QwenModelID]ModelInfo{
	"qwen2.5-coder-32b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.002),
		OutputPrice:         ptr(0.006),
		CacheWritesPrice:    ptr(0.002),
		CacheReadsPrice:     ptr(0.006),
	},
	"qwen2.5-coder-14b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.002),
		OutputPrice:         ptr(0.006),
		CacheWritesPrice:    ptr(0.002),
		CacheReadsPrice:     ptr(0.006),
	},
	"qwen2.5-coder-7b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.001),
		OutputPrice:         ptr(0.002),
		CacheWritesPrice:    ptr(0.001),
		CacheReadsPrice:     ptr(0.002),
	},
	"qwen2.5-coder-3b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.0),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
	"qwen2.5-coder-1.5b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.0),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
	"qwen2.5-coder-0.5b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.0),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
	"qwen-coder-plus-latest": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(3.5),
		OutputPrice:         ptr(7.0),
		CacheWritesPrice:    ptr(3.5),
		CacheReadsPrice:     ptr(7.0),
	},
	"qwen-plus-latest": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.8),
		OutputPrice:         ptr(2.0),
		CacheWritesPrice:    ptr(0.8),
		CacheReadsPrice:     ptr(0.2),
	},
	"qwen-turbo-latest": {
		MaxTokens:           ptr(1000000),
		ContextWindow:       ptr(1000000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.8),
		OutputPrice:         ptr(2.0),
		CacheWritesPrice:    ptr(0.8),
		CacheReadsPrice:     ptr(2.0),
	},
	"qwen-max-latest": {
		MaxTokens:           ptr(30720),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(2.4),
		OutputPrice:         ptr(9.6),
		CacheWritesPrice:    ptr(2.4),
		CacheReadsPrice:     ptr(9.6),
	},
	"qwen-coder-plus": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(3.5),
		OutputPrice:         ptr(7.0),
		CacheWritesPrice:    ptr(3.5),
		CacheReadsPrice:     ptr(7.0),
	},
	"qwen-plus": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.8),
		OutputPrice:         ptr(2.0),
		CacheWritesPrice:    ptr(0.8),
		CacheReadsPrice:     ptr(0.2),
	},
	"qwen-turbo": {
		MaxTokens:           ptr(1000000),
		ContextWindow:       ptr(1000000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.3),
		OutputPrice:         ptr(0.6),
		CacheWritesPrice:    ptr(0.3),
		CacheReadsPrice:     ptr(0.6),
	},
	"qwen-max": {
		MaxTokens:           ptr(30720),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(2.4),
		OutputPrice:         ptr(9.6),
		CacheWritesPrice:    ptr(2.4),
		CacheReadsPrice:     ptr(9.6),
	},
	"deepseek-v3": {
		MaxTokens:           ptr(8000),
		ContextWindow:       ptr(64000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: true,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.28),
		CacheWritesPrice:    ptr(0.14),
		CacheReadsPrice:     ptr(0.014),
	},
	"deepseek-r1": {
		MaxTokens:           ptr(8000),
		ContextWindow:       ptr(64000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: true,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(2.19),
		CacheWritesPrice:    ptr(0.55),
		CacheReadsPrice:     ptr(0.14),
	},
	"qwen-vl-max": {
		MaxTokens:           ptr(30720),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(true),
		SupportsPromptCache: false,
		InputPrice:          ptr(3.0),
		OutputPrice:         ptr(9.0),
		CacheWritesPrice:    ptr(3.0),
		CacheReadsPrice:     ptr(9.0),
	},
	"qwen-vl-max-latest": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(true),
		SupportsPromptCache: false,
		InputPrice:          ptr(3.0),
		OutputPrice:         ptr(9.0),
		CacheWritesPrice:    ptr(3.0),
		CacheReadsPrice:     ptr(9.0),
	},
	"qwen-vl-plus": {
		MaxTokens:           ptr(6000),
		ContextWindow:       ptr(8000),
		SupportsImages:      ptr(true),
		SupportsPromptCache: false,
		InputPrice:          ptr(1.5),
		OutputPrice:         ptr(4.5),
		CacheWritesPrice:    ptr(1.5),
		CacheReadsPrice:     ptr(4.5),
	},
	"qwen-vl-plus-latest": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(true),
		SupportsPromptCache: false,
		InputPrice:          ptr(1.5),
		OutputPrice:         ptr(4.5),
		CacheWritesPrice:    ptr(1.5),
		CacheReadsPrice:     ptr(4.5),
	},
}

// MainlandQwenModels contains information about all available mainland Qwen models
var MainlandQwenModels = map[QwenModelID]ModelInfo{
	"qwen2.5-coder-32b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.002),
		OutputPrice:         ptr(0.006),
		CacheWritesPrice:    ptr(0.002),
		CacheReadsPrice:     ptr(0.006),
	},
	"qwen2.5-coder-14b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.002),
		OutputPrice:         ptr(0.006),
		CacheWritesPrice:    ptr(0.002),
		CacheReadsPrice:     ptr(0.006),
	},
	"qwen2.5-coder-7b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.001),
		OutputPrice:         ptr(0.002),
		CacheWritesPrice:    ptr(0.001),
		CacheReadsPrice:     ptr(0.002),
	},
	"qwen2.5-coder-3b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.0),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
	"qwen2.5-coder-1.5b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.0),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
	"qwen2.5-coder-0.5b-instruct": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.0),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
	"qwen-coder-plus-latest": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(3.5),
		OutputPrice:         ptr(7.0),
		CacheWritesPrice:    ptr(3.5),
		CacheReadsPrice:     ptr(7.0),
	},
	"qwen-plus-latest": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.8),
		OutputPrice:         ptr(2.0),
		CacheWritesPrice:    ptr(0.8),
		CacheReadsPrice:     ptr(0.2),
	},
	"qwen-turbo-latest": {
		MaxTokens:           ptr(1000000),
		ContextWindow:       ptr(1000000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.8),
		OutputPrice:         ptr(2.0),
		CacheWritesPrice:    ptr(0.8),
		CacheReadsPrice:     ptr(2.0),
	},
	"qwen-max-latest": {
		MaxTokens:           ptr(30720),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(2.4),
		OutputPrice:         ptr(9.6),
		CacheWritesPrice:    ptr(2.4),
		CacheReadsPrice:     ptr(9.6),
	},
	"qwq-plus-latest": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(131071),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.0),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
	"qwq-plus": {
		MaxTokens:           ptr(8192),
		ContextWindow:       ptr(131071),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.0),
		CacheWritesPrice:    ptr(0.0),
		CacheReadsPrice:     ptr(0.0),
	},
	"qwen-coder-plus": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(3.5),
		OutputPrice:         ptr(7.0),
		CacheWritesPrice:    ptr(3.5),
		CacheReadsPrice:     ptr(7.0),
	},
	"qwen-plus": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.8),
		OutputPrice:         ptr(2.0),
		CacheWritesPrice:    ptr(0.8),
		CacheReadsPrice:     ptr(0.2),
	},
	"qwen-turbo": {
		MaxTokens:           ptr(1000000),
		ContextWindow:       ptr(1000000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(0.3),
		OutputPrice:         ptr(0.6),
		CacheWritesPrice:    ptr(0.3),
		CacheReadsPrice:     ptr(0.6),
	},
	"qwen-max": {
		MaxTokens:           ptr(30720),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(false),
		SupportsPromptCache: false,
		InputPrice:          ptr(2.4),
		OutputPrice:         ptr(9.6),
		CacheWritesPrice:    ptr(2.4),
		CacheReadsPrice:     ptr(9.6),
	},
	"deepseek-v3": {
		MaxTokens:           ptr(8000),
		ContextWindow:       ptr(64000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: true,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(0.28),
		CacheWritesPrice:    ptr(0.14),
		CacheReadsPrice:     ptr(0.014),
	},
	"deepseek-r1": {
		MaxTokens:           ptr(8000),
		ContextWindow:       ptr(64000),
		SupportsImages:      ptr(false),
		SupportsPromptCache: true,
		InputPrice:          ptr(0.0),
		OutputPrice:         ptr(2.19),
		CacheWritesPrice:    ptr(0.55),
		CacheReadsPrice:     ptr(0.14),
	},
	"qwen-vl-max": {
		MaxTokens:           ptr(30720),
		ContextWindow:       ptr(32768),
		SupportsImages:      ptr(true),
		SupportsPromptCache: false,
		InputPrice:          ptr(3.0),
		OutputPrice:         ptr(9.0),
		CacheWritesPrice:    ptr(3.0),
		CacheReadsPrice:     ptr(9.0),
	},
	"qwen-vl-max-latest": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(true),
		SupportsPromptCache: false,
		InputPrice:          ptr(3.0),
		OutputPrice:         ptr(9.0),
		CacheWritesPrice:    ptr(3.0),
		CacheReadsPrice:     ptr(9.0),
	},
	"qwen-vl-plus": {
		MaxTokens:           ptr(6000),
		ContextWindow:       ptr(8000),
		SupportsImages:      ptr(true),
		SupportsPromptCache: false,
		InputPrice:          ptr(1.5),
		OutputPrice:         ptr(4.5),
		CacheWritesPrice:    ptr(1.5),
		CacheReadsPrice:     ptr(4.5),
	},
	"qwen-vl-plus-latest": {
		MaxTokens:           ptr(129024),
		ContextWindow:       ptr(131072),
		SupportsImages:      ptr(true),
		SupportsPromptCache: false,
		InputPrice:          ptr(1.5),
		OutputPrice:         ptr(4.5),
		CacheWritesPrice:    ptr(1.5),
		CacheReadsPrice:     ptr(4.5),
	},
}

// Helper function to create pointers to values
func ptr[T any](v T) *T {
	return &v
}
