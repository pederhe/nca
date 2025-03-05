package core

import (
	"regexp"
	"strings"
)

// ParseToolUse parses tool use request from AI response
func ParseToolUse(content string) map[string]interface{} {
	// Check for attempt_completion first as a special case
	attemptCompletionRegex := regexp.MustCompile(`<attempt_completion>([\s\S]*?)</attempt_completion>`)
	attemptCompletionMatch := attemptCompletionRegex.FindStringSubmatch(content)
	if len(attemptCompletionMatch) > 0 {
		// Extract result content if available
		resultRegex := regexp.MustCompile(`<result>([\s\S]*?)</result>`)
		resultMatch := resultRegex.FindStringSubmatch(content)

		params := map[string]interface{}{
			"tool": "attempt_completion",
		}

		if len(resultMatch) > 1 {
			params["result"] = resultMatch[1]
		}

		return params
	}

	// Find the first tool tag
	toolNameRegex := regexp.MustCompile(`<([a-zA-Z_]+)>\s*`)
	toolNameMatch := toolNameRegex.FindStringSubmatch(content)
	if len(toolNameMatch) < 2 {
		return nil
	}

	toolName := toolNameMatch[1]

	// Skip if it's attempt_completion (already handled above)
	if toolName == "attempt_completion" {
		return nil
	}

	// Extract the entire tool block
	toolBlockRegex := regexp.MustCompile(`<` + toolName + `>([\s\S]*?)</` + toolName + `>`)
	toolBlockMatch := toolBlockRegex.FindStringSubmatch(content)
	if len(toolBlockMatch) < 2 {
		return nil
	}

	toolBlock := toolBlockMatch[1]

	// Parse parameters
	params := map[string]interface{}{
		"tool": toolName,
	}

	// Find all parameters - using a more robust approach
	// Look for parameter tags directly
	pathMatch := regexp.MustCompile(`<path>([\s\S]*?)</path>`).FindStringSubmatch(toolBlock)
	if len(pathMatch) > 1 {
		params["path"] = strings.TrimSpace(pathMatch[1])
	}

	recursiveMatch := regexp.MustCompile(`<recursive>([\s\S]*?)</recursive>`).FindStringSubmatch(toolBlock)
	if len(recursiveMatch) > 1 {
		recursiveValue := strings.TrimSpace(recursiveMatch[1])
		params["recursive"] = recursiveValue == "true"
	}

	// Handle other parameters based on tool type
	switch toolName {
	case "execute_command":
		commandMatch := regexp.MustCompile(`<command>([\s\S]*?)</command>`).FindStringSubmatch(toolBlock)
		if len(commandMatch) > 1 {
			params["command"] = strings.TrimSpace(commandMatch[1])
		}

		requiresApprovalMatch := regexp.MustCompile(`<requires_approval>([\s\S]*?)</requires_approval>`).FindStringSubmatch(toolBlock)
		if len(requiresApprovalMatch) > 1 {
			approvalValue := strings.TrimSpace(requiresApprovalMatch[1])
			params["requires_approval"] = approvalValue == "true"
		}

	case "read_file":
		// path is already handled above

	case "write_to_file":
		contentMatch := regexp.MustCompile(`<content>([\s\S]*?)</content>`).FindStringSubmatch(toolBlock)
		if len(contentMatch) > 1 {
			params["content"] = contentMatch[1] // Don't trim content to preserve formatting
		}

	case "replace_in_file":
		diffMatch := regexp.MustCompile(`<diff>([\s\S]*?)</diff>`).FindStringSubmatch(toolBlock)
		if len(diffMatch) > 1 {
			params["diff"] = diffMatch[1] // Don't trim diff to preserve formatting
		}

	case "search_files":
		regexMatch := regexp.MustCompile(`<regex>([\s\S]*?)</regex>`).FindStringSubmatch(toolBlock)
		if len(regexMatch) > 1 {
			params["regex"] = strings.TrimSpace(regexMatch[1])
		}

		filePatternMatch := regexp.MustCompile(`<file_pattern>([\s\S]*?)</file_pattern>`).FindStringSubmatch(toolBlock)
		if len(filePatternMatch) > 1 {
			params["file_pattern"] = strings.TrimSpace(filePatternMatch[1])
		}
	}

	return params
}

// RemoveThinkingTags removes content within <thinking></thinking> tags
func RemoveThinkingTags(content string) string {
	// Find and remove all <thinking>...</thinking> blocks
	thinkingRegex := regexp.MustCompile(`<thinking>[\s\S]*?</thinking>`)
	return thinkingRegex.ReplaceAllString(content, "")
}
