package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yourusername/nca/api"
	"github.com/yourusername/nca/config"
	"github.com/yourusername/nca/core"
)

// Version information, injected by compiler
var (
	Version    = "dev"
	BuildTime  = "unknown"
	CommitHash = "unknown"
)

func main() {
	// Parse command line arguments
	promptFlag := flag.Bool("p", false, "Run a one-time query and exit")
	versionFlag := flag.Bool("v", false, "Show version information")
	flag.Parse()

	// Show version information
	if *versionFlag {
		fmt.Printf("nca version: %s\n", Version)
		fmt.Printf("Build time: %s\n", BuildTime)
		fmt.Printf("Commit hash: %s\n", CommitHash)
		return
	}

	args := flag.Args()

	// Handle config command
	if len(args) > 0 && args[0] == "config" {
		handleConfigCommand(args[1:])
		return
	}

	// Check if there's pipe input
	stat, _ := os.Stdin.Stat()
	hasPipe := (stat.Mode() & os.ModeCharDevice) == 0

	var initialPrompt string
	if len(args) > 0 {
		initialPrompt = args[0]
	}

	// Handle pipe input
	if hasPipe && *promptFlag {
		reader := bufio.NewReader(os.Stdin)
		content, _ := io.ReadAll(reader)
		if initialPrompt == "" {
			initialPrompt = string(content)
		} else {
			initialPrompt = initialPrompt + "\n\n" + string(content)
		}
	}

	// Run REPL or one-off query
	if *promptFlag {
		runOneOffQuery(initialPrompt)
	} else {
		runREPL(initialPrompt)
	}
}

// Handle config command
func handleConfigCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: nca config [set|unset] [--global] [key] [value]")
		return
	}

	isGlobal := false
	cmdArgs := args

	// Check for --global flag
	for i, arg := range args {
		if arg == "--global" {
			isGlobal = true
			cmdArgs = append(args[:i], args[i+1:]...)
			break
		}
	}

	switch cmdArgs[0] {
	case "set":
		if len(cmdArgs) < 3 {
			fmt.Println("Usage: nca config set [--global] [key] [value]")
			return
		}
		config.Set(cmdArgs[1], cmdArgs[2], isGlobal)
		fmt.Printf("Set %s = %s\n", cmdArgs[1], cmdArgs[2])
	case "unset":
		if len(cmdArgs) < 2 {
			fmt.Println("Usage: nca config unset [--global] [key]")
			return
		}
		config.Unset(cmdArgs[1], isGlobal)
		fmt.Printf("Removed setting %s\n", cmdArgs[1])
	default:
		fmt.Println("Unknown config command. Available commands: set, unset")
	}
}

// Run interactive REPL
func runREPL(initialPrompt string) {
	conversation := []map[string]string{}

	// If there's an initial prompt, handle it first
	if initialPrompt != "" {
		handlePrompt(initialPrompt, &conversation)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		// Handle slash command
		if strings.HasPrefix(input, "/") {
			handleSlashCommand(input, &conversation)
			continue
		}

		handlePrompt(input, &conversation)
	}
}

// Run one-off query
func runOneOffQuery(prompt string) {
	conversation := []map[string]string{}
	handlePrompt(prompt, &conversation)
}

// Handle user input prompt
func handlePrompt(prompt string, conversation *[]map[string]string) {
	// Add user message to conversation history
	*conversation = append(*conversation, map[string]string{
		"role":    "user",
		"content": prompt,
	})

	// Call API
	response := callAPI(*conversation)

	// Output AI response
	fmt.Println(response.Content)

	// Check for tool use request
	if toolUse := extractToolUse(response.Content); toolUse != nil {
		// Handle tool use
		result := handleToolUse(toolUse)

		// Format tool description based on tool type
		toolDesc := formatToolDescription(toolUse)

		// Add tool result to conversation history with description
		toolResultContent := fmt.Sprintf("%s Result:\n%s", toolDesc, result)
		*conversation = append(*conversation, map[string]string{
			"role":    "user",
			"content": toolResultContent,
		})

		// Call API again to handle tool result
		response = callAPI(*conversation)
		fmt.Println(response.Content)
	}

	// Add AI response to conversation history
	*conversation = append(*conversation, map[string]string{
		"role":    "assistant",
		"content": response.Content,
	})
}

// Format tool description based on tool type and parameters
func formatToolDescription(toolUse map[string]interface{}) string {
	toolName, _ := toolUse["tool"].(string)

	switch toolName {
	case "execute_command":
		command, _ := toolUse["command"].(string)
		return fmt.Sprintf("[%s for '%s']", toolName, command)

	case "read_file", "write_to_file", "replace_in_file", "list_files", "list_code_definition_names":
		path, _ := toolUse["path"].(string)
		return fmt.Sprintf("[%s for '%s']", toolName, path)

	case "search_files":
		regex, _ := toolUse["regex"].(string)
		filePattern, hasPattern := toolUse["file_pattern"].(string)

		if hasPattern && filePattern != "" {
			return fmt.Sprintf("[%s for '%s' in '%s']", toolName, regex, filePattern)
		}
		return fmt.Sprintf("[%s for '%s']", toolName, regex)

	default:
		return fmt.Sprintf("[%s]", toolName)
	}
}

// Handle slash command
func handleSlashCommand(cmd string, conversation *[]map[string]string) {
	switch cmd {
	case "/clear":
		*conversation = []map[string]string{}
		fmt.Println("Conversation history cleared")
	case "/help":
		fmt.Println("Available commands:")
		fmt.Println("/clear - Clear conversation history")
		fmt.Println("/help - Show help information")
	default:
		fmt.Println("Unknown command. Enter /help for help")
	}
}

// API response structure
type APIResponse struct {
	Content string `json:"content"`
}

// Call AI API
func callAPI(conversation []map[string]string) APIResponse {
	// Build system prompt
	systemPrompt, err := core.BuildSystemPrompt()
	if err != nil {
		return APIResponse{
			Content: fmt.Sprintf("Error building system prompt: %s", err),
		}
	}

	// Prepare messages
	messages := []api.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// Add conversation history
	for _, msg := range conversation {
		messages = append(messages, api.Message{
			Role:    msg["role"],
			Content: msg["content"],
		})
	}

	// Create API client
	client := api.NewClient()

	// Call API
	content, err := client.Chat(messages)
	if err != nil {
		return APIResponse{
			Content: fmt.Sprintf("API call error: %s", err),
		}
	}

	// Remove <thinking></thinking> tags from the response
	cleanedContent := core.RemoveThinkingTags(content)

	return APIResponse{
		Content: cleanedContent,
	}
}

// Extract tool use request from AI response
func extractToolUse(content string) map[string]interface{} {
	return core.ParseToolUse(content)
}

// Handle tool use request
func handleToolUse(toolUse map[string]interface{}) string {
	toolName, ok := toolUse["tool"].(string)
	if !ok {
		return "Error: Unable to determine tool to use"
	}

	// Special handling for attempt_completion
	if toolName == "attempt_completion" {
		// Extract result content if available
		if result, ok := toolUse["result"].(string); ok && result != "" {
			return result
		}
		return "Completion finished successfully."
	}

	switch toolName {
	case "execute_command":
		return core.ExecuteCommand(toolUse)
	case "read_file":
		return core.ReadFile(toolUse)
	case "write_to_file":
		return core.WriteToFile(toolUse)
	case "replace_in_file":
		return core.ReplaceInFile(toolUse)
	case "search_files":
		return core.SearchFiles(toolUse)
	case "list_files":
		return core.ListFiles(toolUse)
	case "list_code_definition_names":
		return core.ListCodeDefinitionNames(toolUse)
	default:
		return fmt.Sprintf("Error: Unknown tool '%s'", toolName)
	}
}
