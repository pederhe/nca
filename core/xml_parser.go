package core

import (
	"regexp"
	"sort"
	"strings"
)

// ParseToolUse parses tool use request from AI response
func ParseToolUse(content string) map[string]interface{} {
	// Define the list of root tool tags
	rootTools := []string{
		"execute_command",
		"read_file",
		"write_to_file",
		"replace_in_file",
		"search_files",
		"list_files",
		"list_code_definition_names",
		"attempt_completion",
		"ask_followup_question",
		"plan_mode_response",
		"git_commit",
		"fetch_web_content",
	}

	// Find all root tool tags
	var allToolMatches []struct {
		toolName string
		match    string
		position int
	}

	for _, toolName := range rootTools {
		// Create a complete start and end tag match for each tool name
		fullTagRegex := regexp.MustCompile(`<` + toolName + `>[\s\S]*?</` + toolName + `>`)
		matches := fullTagRegex.FindAllStringSubmatchIndex(content, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				matchText := content[match[0]:match[1]]
				allToolMatches = append(allToolMatches, struct {
					toolName string
					match    string
					position int
				}{
					toolName: toolName,
					match:    matchText,
					position: match[0],
				})
			}
		}
	}

	// Sort tools by position
	sort.Slice(allToolMatches, func(i, j int) bool {
		return allToolMatches[i].position < allToolMatches[j].position
	})

	// Check if there are multiple tool uses
	hasMultipleTools := len(allToolMatches) > 1

	// Find the first tool tag
	var toolName string
	var toolBlock string

	if len(allToolMatches) > 0 {
		toolName = allToolMatches[0].toolName

		// Extract the entire tool block
		toolBlockRegex := regexp.MustCompile(`<` + toolName + `>([\s\S]*?)</` + toolName + `>`)
		toolBlockMatch := toolBlockRegex.FindStringSubmatch(allToolMatches[0].match)
		if len(toolBlockMatch) < 2 {
			return nil
		}
		toolBlock = toolBlockMatch[1]
	} else {
		// If no tool tags found, return nil
		return nil
	}

	// Parse parameters
	params := map[string]interface{}{
		"tool": toolName,
	}

	// Add flag for multiple tools detection
	if hasMultipleTools {
		// Extract all tool names for the warning message
		var toolNames []string
		for _, match := range allToolMatches {
			toolNames = append(toolNames, match.toolName)
		}
		params["has_multiple_tools"] = true
		params["detected_tools"] = strings.Join(toolNames, ", ")
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

	case "git_commit":
		// Extract message parameter - required
		messageMatch := regexp.MustCompile(`<message>([\s\S]*?)</message>`).FindStringSubmatch(toolBlock)
		if len(messageMatch) > 1 {
			params["message"] = strings.TrimSpace(messageMatch[1])
		}

		// Extract files parameter - required
		filesMatch := regexp.MustCompile(`<files>([\s\S]*?)</files>`).FindStringSubmatch(toolBlock)
		if len(filesMatch) > 1 {
			filesContent := strings.TrimSpace(filesMatch[1])
			if filesContent != "" {
				// Split by newlines and trim each line
				filesList := []string{}
				for _, file := range strings.Split(filesContent, "\n") {
					trimmedFile := strings.TrimSpace(file)
					if trimmedFile != "" {
						filesList = append(filesList, trimmedFile)
					}
				}
				params["files"] = filesList
			}
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

	case "fetch_web_content":
		urlMatch := regexp.MustCompile(`<url>([\s\S]*?)</url>`).FindStringSubmatch(toolBlock)
		if len(urlMatch) > 1 {
			params["url"] = strings.TrimSpace(urlMatch[1])
		}

	case "attempt_completion":
		// Extract result content if available
		resultRegex := regexp.MustCompile(`<r>([\s\S]*?)</r>`)
		resultMatch := resultRegex.FindStringSubmatch(toolBlock)
		if len(resultMatch) > 1 {
			params["result"] = resultMatch[1]
		}

		// Extract command if available
		commandRegex := regexp.MustCompile(`<command>([\s\S]*?)</command>`)
		commandMatch := commandRegex.FindStringSubmatch(toolBlock)
		if len(commandMatch) > 1 {
			params["command"] = commandMatch[1]
		}

	case "plan_mode_response":
		responseMatch := regexp.MustCompile(`<response>([\s\S]*?)</response>`).FindStringSubmatch(toolBlock)
		if len(responseMatch) > 1 {
			params["response"] = responseMatch[1]
		}
	}

	return params
}

// RemoveThinkingTags removes content within <thinking></thinking> tags
func RemoveThinkingTags(content string) string {
	// Use a more complex method to handle nested tags
	// First find the outermost thinking tags
	result := content

	// Handle nested tags by removing from the innermost outward
	// Repeatedly search and replace until no more thinking tags remain
	for {
		// Find the innermost thinking tags (thinking tags that don't contain other thinking tags)
		innerThinkingRegex := regexp.MustCompile(`<thinking>[^<]*?</thinking>`)
		if !innerThinkingRegex.MatchString(result) {
			// If no innermost tags found, try to find thinking tags that may contain other content
			outerThinkingRegex := regexp.MustCompile(`<thinking>[\s\S]*?</thinking>`)
			if !outerThinkingRegex.MatchString(result) {
				// If no thinking tags found at all, exit the loop
				break
			}
			// Replace the found tags
			result = outerThinkingRegex.ReplaceAllString(result, "")
		} else {
			// Replace the innermost tags found
			result = innerThinkingRegex.ReplaceAllString(result, "")
		}
	}

	return result
}
