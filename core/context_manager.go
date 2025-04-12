package core

import (
	"math"
	"strings"

	"github.com/pederhe/nca/api/types"
)

// UpdateContextMessages updates the context messages if the total tokens exceed the max allowed size
func UpdateContextMessages(modelInfo *types.ModelInfo, conversation *[]map[string]string, currentDeletedRange *[2]int, previousUsage *types.Usage) bool {
	_, maxAllowedSize := getContextWindowInfo(modelInfo)
	if previousUsage != nil && previousUsage.TotalTokens >= maxAllowedSize {
		keep := "half"
		if previousUsage.TotalTokens/2 > maxAllowedSize {
			keep = "quarter"
		}
		newRange := GetNextTruncationRange(*conversation, *currentDeletedRange, keep)
		// If we can't truncate any more messages, exit
		if newRange[1] <= newRange[0] {
			return false
		}

		// Update current deleted range
		*currentDeletedRange = newRange

		// Create new conversation slice with truncated messages
		// Keep messages before the truncation range and after the truncation range
		*conversation = append((*conversation)[:newRange[0]], (*conversation)[newRange[1]+1:]...)

		return true
	}

	return false
}

// GetNextTruncationRange calculates the range of messages to be removed from the conversation history
func GetNextTruncationRange(conversation []map[string]string, currentDeletedRange [2]int, keep string) [2]int {
	// Always keep the first message
	const rangeStartIndex = 1
	startOfRest := 1
	if currentDeletedRange[1] > 0 {
		startOfRest = currentDeletedRange[1] + 1
	}

	var messagesToRemove int
	if keep == "half" {
		// Remove half of remaining user-assistant pairs
		messagesToRemove = (len(conversation) - startOfRest) / 2
		// Ensure we remove an even number of messages to maintain user-assistant pairs
		messagesToRemove = (messagesToRemove / 2) * 2
	} else {
		// Remove 3/4 of remaining user-assistant pairs
		messagesToRemove = ((len(conversation) - startOfRest) * 3) / 4
		// Ensure we remove an even number of messages to maintain user-assistant pairs
		messagesToRemove = (messagesToRemove / 2) * 2
	}

	rangeEndIndex := startOfRest + messagesToRemove - 1

	// Make sure the last message being removed is a user message
	// This preserves the user-assistant-user-assistant structure
	if rangeEndIndex < len(conversation) && conversation[rangeEndIndex]["role"] != "user" {
		rangeEndIndex--
	}

	return [2]int{rangeStartIndex, rangeEndIndex}
}

// getContextWindowInfo returns the context window and max allowed size for a given models
func getContextWindowInfo(modelInfo *types.ModelInfo) (int, int) {
	// Get context window from model info, default to 128000 if not specified
	contextWindow := 128000
	if modelInfo != nil && modelInfo.ContextWindow != nil {
		contextWindow = *modelInfo.ContextWindow
	}

	// Handle special cases like DeepSeek
	if strings.Contains(strings.ToLower(modelInfo.Name), "deepseek") {
		contextWindow = 64000
	}

	var maxAllowedSize int
	switch contextWindow {
	case 64000: // deepseek models
		maxAllowedSize = contextWindow - 27000
	case 128000: // most models
		maxAllowedSize = contextWindow - 30000
	case 200000: // claude models
		maxAllowedSize = contextWindow - 40000
	default:
		// For other models, use 80% of context window or contextWindow - 40000, whichever is larger
		maxAllowedSize = int(math.Max(float64(contextWindow-40000), float64(contextWindow)*0.8))
	}

	return contextWindow, maxAllowedSize
}
