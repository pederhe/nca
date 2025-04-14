package core

import (
	"testing"

	"github.com/pederhe/nca/pkg/api/types"
)

func TestGetContextWindowInfo(t *testing.T) {
	tests := []struct {
		name           string
		modelInfo      *types.ModelInfo
		expectedWindow int
		expectedSize   int
	}{
		{
			name: "DeepSeek model",
			modelInfo: &types.ModelInfo{
				Name:          "deepseek-coder",
				ContextWindow: nil,
			},
			expectedWindow: 64000,
			expectedSize:   37000, // 64000 - 27000
		},
		{
			name: "Standard model with 128k context",
			modelInfo: &types.ModelInfo{
				Name:          "gpt-4",
				ContextWindow: intPtr(128000),
			},
			expectedWindow: 128000,
			expectedSize:   98000, // 128000 - 30000
		},
		{
			name: "Claude model",
			modelInfo: &types.ModelInfo{
				Name:          "claude-3",
				ContextWindow: intPtr(200000),
			},
			expectedWindow: 200000,
			expectedSize:   160000, // 200000 - 40000
		},
		{
			name: "Custom model with 100k context",
			modelInfo: &types.ModelInfo{
				Name:          "custom-model",
				ContextWindow: intPtr(100000),
			},
			expectedWindow: 100000,
			expectedSize:   80000, // max(100000-40000, 100000*0.8)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window, size := getContextWindowInfo(tt.modelInfo)
			if window != tt.expectedWindow {
				t.Errorf("getContextWindowInfo() window = %v, want %v", window, tt.expectedWindow)
			}
			if size != tt.expectedSize {
				t.Errorf("getContextWindowInfo() size = %v, want %v", size, tt.expectedSize)
			}
		})
	}
}

func TestGetNextTruncationRange(t *testing.T) {
	tests := []struct {
		name                string
		conversation        []map[string]string
		currentDeletedRange [2]int
		keep                string
		expectedRange       [2]int
	}{
		{
			name: "Half keep with even messages",
			conversation: []map[string]string{
				{"role": "system", "content": "system message"},
				{"role": "user", "content": "user1"},
				{"role": "assistant", "content": "assistant1"},
				{"role": "user", "content": "user2"},
				{"role": "assistant", "content": "assistant2"},
			},
			currentDeletedRange: [2]int{0, 0},
			keep:                "half",
			expectedRange:       [2]int{1, 1}, // Keep half, delete 1 message pair starting from index 1
		},
		{
			name: "Quarter keep with even messages",
			conversation: []map[string]string{
				{"role": "system", "content": "system message"},
				{"role": "user", "content": "user1"},
				{"role": "assistant", "content": "assistant1"},
				{"role": "user", "content": "user2"},
				{"role": "assistant", "content": "assistant2"},
				{"role": "user", "content": "user3"},
				{"role": "assistant", "content": "assistant3"},
				{"role": "user", "content": "user4"},
				{"role": "assistant", "content": "assistant4"},
			},
			currentDeletedRange: [2]int{0, 0},
			keep:                "quarter",
			expectedRange:       [2]int{1, 5}, // Keep quarter, delete 5 message pairs starting from index 1
		},
		{
			name: "Half keep with odd messages",
			conversation: []map[string]string{
				{"role": "system", "content": "system message"},
				{"role": "user", "content": "user1"},
				{"role": "assistant", "content": "assistant1"},
				{"role": "user", "content": "user2"},
				{"role": "assistant", "content": "assistant2"},
				{"role": "user", "content": "user3"},
			},
			currentDeletedRange: [2]int{0, 0},
			keep:                "half",
			expectedRange:       [2]int{1, 1}, // Keep half, delete 1 message pair starting from index 1
		},
		{
			name: "With previous deletion",
			conversation: []map[string]string{
				{"role": "system", "content": "system message"},
				{"role": "user", "content": "user1"},
				{"role": "assistant", "content": "assistant1"},
				{"role": "user", "content": "user2"},
				{"role": "assistant", "content": "assistant2"},
			},
			currentDeletedRange: [2]int{1, 2},
			keep:                "half",
			expectedRange:       [2]int{1, 1}, // Keep half, delete 1 message pair starting from index 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetNextTruncationRange(tt.conversation, tt.currentDeletedRange, tt.keep)
			if got != tt.expectedRange {
				t.Errorf("GetNextTruncationRange() = %v, want %v", got, tt.expectedRange)
			}
		})
	}
}

// Manual test for UpdateContextMessages function to help us understand its correct behavior
func TestUpdateContextMessagesManually(t *testing.T) {
	// 1. Set up a GPT-4 model, simulating a scenario exceeding the max allowed size
	modelInfo := &types.ModelInfo{
		Name:          "gpt-4",
		ContextWindow: intPtr(128000),
	}
	conversation := []map[string]string{
		{"role": "system", "content": "system message"},
		{"role": "user", "content": "user1"},
		{"role": "assistant", "content": "assistant1"},
		{"role": "user", "content": "user2"},
		{"role": "assistant", "content": "assistant2"},
		{"role": "user", "content": "user3"},
		{"role": "assistant", "content": "assistant3"},
		{"role": "user", "content": "user4"},
		{"role": "assistant", "content": "assistant4"},
		{"role": "user", "content": "user5"},
		{"role": "assistant", "content": "assistant5"},
		{"role": "user", "content": "user6"},
		{"role": "assistant", "content": "assistant6"},
	}
	currentDeletedRange := [2]int{0, 0}

	// Output initial state
	t.Logf("Initial conversation length: %d", len(conversation))
	t.Logf("Initial deleted range: %v", currentDeletedRange)

	// Calculate maxAllowedSize
	_, maxAllowedSize := getContextWindowInfo(modelInfo)
	t.Logf("Max allowed size: %d", maxAllowedSize)

	// Set TotalTokens far exceeding maxAllowedSize
	previousUsage := &types.Usage{TotalTokens: maxAllowedSize + 10000}
	t.Logf("Total tokens: %d", previousUsage.TotalTokens)

	// Call the function
	result := UpdateContextMessages(modelInfo, &conversation, &currentDeletedRange, previousUsage)

	// Output results
	t.Logf("Result: %v", result)
	t.Logf("Final conversation length: %d", len(conversation))
	t.Logf("Final deleted range: %v", currentDeletedRange)
	t.Logf("Messages remaining:")
	for i, msg := range conversation {
		t.Logf("Msg %d: %s - %s", i, msg["role"], msg["content"])
	}
}

func TestUpdateContextMessages(t *testing.T) {
	tests := []struct {
		name                string
		modelInfo           *types.ModelInfo
		conversation        []map[string]string
		currentDeletedRange [2]int
		previousUsage       *types.Usage
		expectedResult      bool
		expectedLength      int
	}{
		{
			name: "No truncation needed",
			modelInfo: &types.ModelInfo{
				Name:          "gpt-4",
				ContextWindow: intPtr(128000),
			},
			conversation: []map[string]string{
				{"role": "system", "content": "system message"},
				{"role": "user", "content": "user1"},
				{"role": "assistant", "content": "assistant1"},
			},
			currentDeletedRange: [2]int{0, 0},
			previousUsage:       &types.Usage{TotalTokens: 1000},
			expectedResult:      false,
			expectedLength:      3,
		},
		{
			name: "Truncation needed but not enough messages to truncate",
			modelInfo: &types.ModelInfo{
				Name:          "gpt-4",
				ContextWindow: intPtr(128000),
			},
			conversation: []map[string]string{
				{"role": "system", "content": "system message"},
				{"role": "user", "content": "user1"},
				{"role": "assistant", "content": "assistant1"},
				{"role": "user", "content": "user2"},
				{"role": "assistant", "content": "assistant2"},
			},
			currentDeletedRange: [2]int{0, 0},
			previousUsage:       &types.Usage{TotalTokens: 200000}, // Exceeds limit but returns false
			expectedResult:      false,                             // The condition newRange[1] <= newRange[0] causes a return false
			expectedLength:      5,                                 // No messages will be deleted
		},
		{
			name: "Truncation with enough messages",
			modelInfo: &types.ModelInfo{
				Name:          "gpt-4",
				ContextWindow: intPtr(128000),
			},
			conversation: []map[string]string{
				{"role": "system", "content": "system message"},
				{"role": "user", "content": "user1"},
				{"role": "assistant", "content": "assistant1"},
				{"role": "user", "content": "user2"},
				{"role": "assistant", "content": "assistant2"},
				{"role": "user", "content": "user3"},
				{"role": "assistant", "content": "assistant3"},
				{"role": "user", "content": "user4"},
				{"role": "assistant", "content": "assistant4"},
				{"role": "user", "content": "user5"},
				{"role": "assistant", "content": "assistant5"},
				{"role": "user", "content": "user6"},
				{"role": "assistant", "content": "assistant6"},
			},
			currentDeletedRange: [2]int{0, 0},
			previousUsage:       &types.Usage{TotalTokens: 250000},
			expectedResult:      true,
			expectedLength:      6, // Length after deleting indices 1-5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Copy conversation to avoid modifying original test data
			conversation := make([]map[string]string, len(tt.conversation))
			copy(conversation, tt.conversation)
			currentDeletedRange := tt.currentDeletedRange

			// Calculate and print maxAllowedSize
			_, maxAllowedSize := getContextWindowInfo(tt.modelInfo)
			t.Logf("Test: %s - Max allowed size: %d, Total tokens: %d",
				tt.name, maxAllowedSize, tt.previousUsage.TotalTokens)

			// Calculate expected truncation range
			newRange := GetNextTruncationRange(conversation, currentDeletedRange, "half")
			t.Logf("Truncation range: %v", newRange)

			// Call the function
			got := UpdateContextMessages(tt.modelInfo, &conversation, &currentDeletedRange, tt.previousUsage)

			// Print results
			t.Logf("Result: %v, Conversation length: %d, Range: %v",
				got, len(conversation), currentDeletedRange)

			// Check results
			if got != tt.expectedResult {
				t.Errorf("UpdateContextMessages() = %v, want %v", got, tt.expectedResult)
			}
			if len(conversation) != tt.expectedLength {
				t.Errorf("UpdateContextMessages() conversation length = %v, want %v", len(conversation), tt.expectedLength)
				// Print conversation content
				for i, msg := range conversation {
					t.Logf("Msg %d: %s - %s", i, msg["role"], msg["content"])
				}
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
