package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/pederhe/nca/api"
	"github.com/pederhe/nca/api/types"
	"github.com/pederhe/nca/config"
	"github.com/pederhe/nca/core"
	"github.com/pederhe/nca/utils"
)

// Version information, injected by compiler
var (
	Version    = "dev"
	BuildTime  = "unknown"
	CommitHash = "unknown"
)

// Debug mode variables
var (
	debugMode    bool
	debugLogFile *os.File
	sessionID    string
	debugLogPath string
)

func main() {
	// Set custom usage function for flag package
	flag.Usage = func() {
		displayHelp()
	}

	// Parse command line arguments
	promptFlag := flag.Bool("p", false, "Run a one-time query and exit")
	versionFlag := flag.Bool("v", false, "Show version information")
	debugFlag := flag.Bool("debug", false, "Enable debug mode to log conversation data")
	flag.Parse()

	// Show version information
	if *versionFlag {
		fmt.Printf("NCA version: %s\n", Version)
		fmt.Printf("Build time: %s\n", BuildTime)
		fmt.Printf("Commit hash: %s\n", CommitHash)
		return
	}

	// Initialize debug mode if enabled
	if *debugFlag {
		debugMode = true
		initDebugMode()
		defer closeDebugLog()
		logDebug("Program started with debug mode enabled\n")
	}

	args := flag.Args()

	// Process command line arguments
	if len(args) > 0 {
		switch args[0] {
		case "config":
			// Handle configuration settings command
			logDebug(fmt.Sprintf("Config command: %v\n", args))
			handleConfigCommand(args[1:])
			return
		case "commit":
			// Handle git commit operation command
			logDebug("Commit command detected\n")
			runREPL("commit all current changes, and summarize the changes")
			return
		case "help":
			// Display help information
			logDebug("Help command detected\n")
			displayHelp()
			return
		}
	}

	// Check if there's pipe input
	stat, _ := os.Stdin.Stat()
	hasPipe := (stat.Mode() & os.ModeCharDevice) == 0

	// Prepare initial prompt from args
	var initialPrompt string
	if len(args) > 0 {
		initialPrompt = strings.Join(args, " ")
	}

	// Handle pipe input
	if hasPipe {
		logDebug("Detected pipe input\n")
		// Read from pipe
		reader := bufio.NewReader(os.Stdin)
		content, err := io.ReadAll(reader)
		if err != nil {
			fmt.Println("Error reading from pipe:", err)
			logDebug(fmt.Sprintf("Error reading from pipe: %s\n", err))
			return
		}

		logDebug(fmt.Sprintf("Pipe input content length: %d bytes\n", len(content)))

		// Combine pipe content with initial prompt if any
		if initialPrompt == "" {
			initialPrompt = string(content)
		} else {
			initialPrompt = initialPrompt + "\n\n" + string(content)
		}
	}

	// Run REPL or one-off query
	if *promptFlag {
		if initialPrompt == "" {
			fmt.Println("Error: No prompt provided for one-time query")
			logDebug("Error: No prompt provided for one-time query\n")
			return
		}
		logDebug(fmt.Sprintf("One-time query mode with prompt: %s\n", initialPrompt))
		runOneOffQuery(initialPrompt)
	} else {
		logDebug("Starting interactive REPL mode\n")
		if initialPrompt != "" {
			logDebug(fmt.Sprintf("With initial prompt: %s\n", initialPrompt))
		}
		runREPL(initialPrompt)
	}
}

// Handle config command
func handleConfigCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: nca config [set|unset|list] [--global] [key] [value]")
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
	case "list":
		// Get all configuration values
		allConfigs := config.GetAll()

		if len(allConfigs) == 0 {
			fmt.Println("No configuration settings found.")
			return
		}

		fmt.Println("Current configuration settings:")
		fmt.Println("------------------------------")
		for key, value := range allConfigs {
			fmt.Printf("%s = %s\n", key, value)
		}
		fmt.Println("------------------------------")
	default:
		fmt.Println("Unknown config command. Available commands: set, unset, list")
	}
}

// Run interactive REPL
func runREPL(initialPrompt string) {
	conversation := []map[string]string{}

	// Log REPL start in debug mode
	logDebug("Starting REPL session\n")
	if initialPrompt != "" {
		logDebug(fmt.Sprintf("Initial prompt: %s\n", initialPrompt))
	}

	// If there's an initial prompt, handle it first
	if initialPrompt != "" {
		handlePrompt(initialPrompt, &conversation)
	} else {
		fmt.Printf("NCA %s (%s,%s)\n", Version, BuildTime, CommitHash)
		fmt.Println("Type /help for help")
		if debugMode {
			fmt.Printf(utils.ColorYellow+"Debug mode enabled. Logs saved to: %s\n"+utils.ColorReset, debugLogPath)
		}
	}

	// Create custom completer for commands
	completer := readline.NewPrefixCompleter(
		readline.PcItem("/clear"),
		readline.PcItem("/help"),
		readline.PcItem("/exit"),
		readline.PcItem("/quit"),
	)

	// Initialize readline with custom config
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            utils.ColorPurple + ">>> " + utils.ColorReset,
		HistoryFile:       os.Getenv("HOME") + "/.nca_history",
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true, // Case-insensitive history search
		AutoComplete:      completer,
	})
	if err != nil {
		fmt.Println("Error initializing readline:", err)
		logDebug(fmt.Sprintf("Error initializing readline: %s\n", err))
		return
	}
	defer rl.Close()

	for {
		// Read input using readline
		input, err := rl.Readline()
		if err != nil {
			// Handle Ctrl+C or Ctrl+D
			if err == readline.ErrInterrupt {
				fmt.Println("Interrupted")
				logDebug("User interrupted input with Ctrl+C\n")
				continue
			} else if err == io.EOF {
				fmt.Println("Exiting")
				logDebug("User exited with Ctrl+D\n")
				break
			}
			fmt.Println("Error reading input:", err)
			logDebug(fmt.Sprintf("Error reading input: %s\n", err))
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle slash command
		if strings.HasPrefix(input, "/") {
			logDebug(fmt.Sprintf("Slash command: %s\n", input))
			if input == "/exit" || input == "/quit" {
				fmt.Println("Exiting")
				logDebug("User exited with /exit or /quit command\n")
				break
			}
			handleSlashCommand(input, &conversation)
			continue
		}

		handlePrompt(input, &conversation)
	}
}

// Run one-off query
func runOneOffQuery(prompt string) {
	conversation := []map[string]string{}

	logDebug("Running one-off query mode\n")
	logDebug(fmt.Sprintf("Query: %s\n", prompt))

	handlePrompt(prompt, &conversation)

	logDebug("One-off query completed\n")
}

// Handle user input prompt
func handlePrompt(prompt string, conversation *[]map[string]string) {
	// Add user message to conversation history
	*conversation = append(*conversation, map[string]string{
		"role":    "user",
		"content": prompt,
	})

	// Log user input in debug mode
	logDebug(fmt.Sprintf("USER INPUT: %s\n", prompt))

	// Count of consecutive responses without tool use
	noToolUseCount := 0

	// Multi-step task processing loop
	for {
		// Call API
		response, err := callAPI(*conversation)
		if err != nil {
			fmt.Println("Error calling API:", err)
			logDebug(fmt.Sprintf("API ERROR: %s\n", err))

			// Remove the last user message to avoid consecutive user messages
			// 像DeepSeek-R1这样的模型不支持连续的user消息
			if len(*conversation) > 0 && (*conversation)[len(*conversation)-1]["role"] == "user" {
				*conversation = (*conversation)[:len(*conversation)-1]
			}

			break
		}

		// Check if there's a tool use request
		toolUse := extractToolUse(response.Content)

		// Add AI response to conversation history
		*conversation = append(*conversation, map[string]string{
			"role":    "assistant",
			"content": response.Content,
		})

		// Process tool use request
		if toolUse != nil {
			// Reset the counter for responses without tool use
			noToolUseCount = 0

			// Log tool use in debug mode
			toolName, _ := toolUse["tool"].(string)
			logDebug(fmt.Sprintf("TOOL USE: %v\n", toolUse))

			result := handleToolUse(toolUse)

			// Log tool result in debug mode
			logDebug(fmt.Sprintf("TOOL RESULT: %s\n", result))

			// Get tool name (already extracted above)
			// Check if it's the task completion tool
			if toolName == "attempt_completion" {
				fmt.Println(utils.ColorYellow + result + utils.ColorReset)
				// Task completed, exit loop
				break
			}
			if toolName == "plan_mode_response" || toolName == "ask_followup_question" {
				// Task completed, exit loop
				break
			}

			// Format tool description based on tool type
			toolDesc := formatToolDescription(toolUse)

			// Add tool result to conversation history with description
			// 任务最后一个工具使用结果不记录，像DeepSeek-R1这样的模型不支持连续的user消息
			toolResultContent := fmt.Sprintf("%s Result:\n%s", toolDesc, result)
			*conversation = append(*conversation, map[string]string{
				"role":    "user",
				"content": toolResultContent,
			})

			// Continue loop, process next step
		} else {
			// Increment counter for responses without tool use
			noToolUseCount++

			// Check if exceeded 3 attempts without tool use
			if noToolUseCount >= 3 {
				errorMessage := "[FATAL ERROR] You failed to use a tool after 3 attempts. Exiting task."
				//fmt.Println("\n" + errorMessage)
				logDebug(fmt.Sprintf("ERROR: %s\n", errorMessage))
				*conversation = append(*conversation, map[string]string{
					"role":    "user",
					"content": errorMessage,
				})
				break
			}

			// No tool use request, add error message to conversation history
			errorMessage := fmt.Sprintf("[ERROR] You did not use a tool in your previous response! Please retry with a tool use. (Attempt %d/3)", noToolUseCount)
			//fmt.Println("\n" + errorMessage)
			logDebug(fmt.Sprintf("ERROR: %s\n", errorMessage))
			*conversation = append(*conversation, map[string]string{
				"role":    "user",
				"content": errorMessage,
			})
			// Don't exit loop, continue requesting AI to use a tool
		}
	}
}

// Format tool description based on tool type and parameters
func formatToolDescription(toolUse map[string]interface{}) string {
	toolName, _ := toolUse["tool"].(string)

	switch toolName {
	case "attempt_completion":
		return "[attempt_completion]"

	case "plan_mode_response":
		return "[plan_mode_response]"

	case "ask_followup_question":
		question, _ := toolUse["question"].(string)
		return fmt.Sprintf("[%s for '%s']", toolName, question)

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

	case "git_commit":
		message, _ := toolUse["message"].(string)
		files, ok := toolUse["files"].([]string)

		if ok && len(files) > 0 {
			return fmt.Sprintf("[%s for message '%s' with files: %s]", toolName, message, strings.Join(files, ", "))
		}

		return fmt.Sprintf("[%s for message '%s']", toolName, message)

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
		logDebug("Conversation history cleared by user\n")
	case "/help":
		fmt.Println("\nINTERACTIVE COMMANDS:")
		fmt.Println("  /clear  - Clear conversation history")
		fmt.Println("  /exit   - Exit the program")
		fmt.Println("  /quit   - Exit the program")
		fmt.Println("  /help   - Show help information")
		logDebug("Help information displayed\n")
	case "/exit", "/quit":
		// These are handled in the runREPL function
		// Nothing to do here
	default:
		fmt.Println("Unknown command. Enter /help for help")
		logDebug(fmt.Sprintf("Unknown command attempted: %s\n", cmd))
	}
}

// API response structure
type APIResponse struct {
	ReasoningContent string `json:"reasoning_content"`
	Content          string `json:"content"`
}

// Call AI API
func callAPI(conversation []map[string]string) (APIResponse, error) {
	// Build system prompt
	systemPrompt, err := core.BuildSystemPrompt()
	if err != nil {
		logDebug(fmt.Sprintf("ERROR building system prompt: %s\n", err))
		return APIResponse{}, fmt.Errorf("error building system prompt: %s", err)
	}

	// Prepare messages
	messages := []types.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// Add conversation history
	for _, msg := range conversation {
		messages = append(messages, types.Message{
			Role:    msg["role"],
			Content: msg["content"],
		})
	}

	// Log API request in debug mode
	logDebug("API REQUEST PAYLOAD:\n")
	for i, msg := range messages {
		// Truncate system message for brevity in logs
		content := msg.Content
		if msg.Role == "system" && len(content) > 100 {
			content = content[:100] + "... [truncated]"
		}
		logDebug(fmt.Sprintf("  [%d] %s: %s\n", i, msg.Role, content))
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		fmt.Println("Error: Failed to create API client:", err)
		os.Exit(1)
	}

	// Start dynamic loading animation
	stopLoading := make(chan bool, 1)   // Use buffered channel to prevent blocking
	animationDone := make(chan bool, 1) // Channel to confirm animation has stopped
	go showLoadingAnimation(stopLoading, animationDone)

	// Create a filter for XML tags
	filter := core.NewXMLTagFilter()

	// Flag to track if animation has been stopped
	var animationStopped bool = false
	var startReasoning bool = false

	// Define callback function for streaming
	callback := func(reasoningChunk string, chunk string, isDone bool) {
		// Stop loading animation when first response chunk is received
		if (len(reasoningChunk) > 0 || len(chunk) > 0) && !animationStopped {
			stopLoading <- true
			<-animationDone // Wait for animation to actually stop
			animationStopped = true
		}

		if reasoningChunk != "" {
			if !startReasoning {
				startReasoning = true
				fmt.Println(utils.ColorBlue + "Reasoning:" + utils.ColorReset)
			}
			fmt.Print(reasoningChunk)
		} else if chunk != "" {
			if startReasoning {
				fmt.Println(utils.ColorBlue + "\n----------------------------" + utils.ColorReset)
				startReasoning = false
			}
			// Filter and print the chunk
			filtered := filter.ProcessChunk(chunk)
			fmt.Print(filtered)
		}
	}

	// Call API with streaming
	reasoningContent, content, err := client.ChatStream(messages, callback)
	fmt.Println() // Add newline after streaming completes

	// Log raw response in debug mode
	logDebug(fmt.Sprintf("RAW API RESPONSE STREAM:\n%s\n%s\n", reasoningContent, content))

	// Ensure loading animation is stopped
	if !animationStopped {
		stopLoading <- true
		<-animationDone
	}

	if err != nil {
		logDebug(fmt.Sprintf("API STREAM ERROR: %s\n", err))
		return APIResponse{}, fmt.Errorf("API call error: %s", err)
	}

	// Remove <thinking></thinking> tags from the response
	//cleanedContent := core.RemoveThinkingTags(content)
	// The thinking tags here are the AI's thought process, not reasoning process, so we're not removing them yet

	return APIResponse{
		ReasoningContent: reasoningContent,
		Content:          content,
	}, nil
}

// Display loading animation
func showLoadingAnimation(stop chan bool, done chan bool) {
	// Loading animation characters
	spinChars := []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"}
	i := 0

	// Clear current line and display initial message
	fmt.Print("\r\033[KGenerating... ")

	for {
		select {
		case <-stop:
			// Clear animation line to ensure it doesn't affect subsequent output
			fmt.Print("\r\033[K")
			done <- true // Notify that animation has stopped
			return
		default:
			// Display spinning animation
			fmt.Printf("\r\033[KGenerating... %s", spinChars[i])
			i = (i + 1) % len(spinChars)
			time.Sleep(100 * time.Millisecond)
		}
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
		// If there's a command parameter, execute the command
		if commandStr, ok := toolUse["command"].(string); ok && commandStr != "" {
			// Create a temporary tool use request
			cmdToolUse := map[string]interface{}{
				"tool":              "execute_command",
				"command":           commandStr,
				"requires_approval": true,
			}

			// Execute command
			return core.ExecuteCommand(cmdToolUse)
		}

		return ""
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
	case "ask_followup_question":
		return core.FollowupQuestion(toolUse)
	case "plan_mode_response":
		return core.PlanModeResponse(toolUse)
	case "git_commit":
		return core.GitCommit(toolUse)
	default:
		return fmt.Sprintf("Error: Unknown tool '%s'", toolName)
	}
}

// Initialize debug mode, creating necessary directories and log file
func initDebugMode() {
	// Create base debug directory if it doesn't exist
	debugBaseDir := filepath.Join(os.Getenv("HOME"), ".nca", "debug")
	if err := os.MkdirAll(debugBaseDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create debug directory: %s\n", err)
		debugMode = false
		return
	}

	// Create directory for today's date
	now := time.Now()
	dateDir := filepath.Join(debugBaseDir, now.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create date directory: %s\n", err)
		debugMode = false
		return
	}

	// Generate unique session ID based on timestamp
	sessionID = now.Format("150405-") + fmt.Sprintf("%03d", now.Nanosecond()/1000000)

	// Create log file
	debugLogPath = filepath.Join(dateDir, fmt.Sprintf("session_%s.log", sessionID))
	var err error
	debugLogFile, err = os.Create(debugLogPath)
	if err != nil {
		fmt.Printf("Warning: Failed to create debug log file: %s\n", err)
		debugMode = false
		return
	}

	// Log session start
	logDebug(fmt.Sprintf("Session started at %s\n", now.Format(time.RFC3339)))
	logDebug(fmt.Sprintf("NCA version: %s, Build time: %s, Commit hash: %s\n",
		Version, BuildTime, CommitHash))
}

// Write a message to the debug log
func logDebug(message string) {
	if !debugMode || debugLogFile == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, message)

	if _, err := debugLogFile.WriteString(logEntry); err != nil {
		fmt.Printf("Warning: Failed to write to debug log: %s\n", err)
	}
}

// Close the debug log file
func closeDebugLog() {
	if debugLogFile != nil {
		logDebug("Session ended\n")
		debugLogFile.Close()
		debugLogFile = nil
	}
}

// displayHelp shows all available commands and options
func displayHelp() {
	fmt.Println("NCA - Nano Code Agent")
	fmt.Printf("Version: %s, Build time: %s, Commit hash: %s\n\n", Version, BuildTime, CommitHash)

	fmt.Println("USAGE:")
	fmt.Println("  nca [options] [prompt]")
	fmt.Println("  nca [command]")

	fmt.Println("\nCOMMANDS:")
	fmt.Println("  help    - Display this help information")
	fmt.Println("  config  - Manage configuration settings")
	fmt.Println("           Usage: nca config [set|unset|list] [--global] [key] [value]")
	fmt.Println("  commit  - Automatically commit all current changes, and summarize the changes")

	fmt.Println("\nOPTIONS:")
	fmt.Println("  -p      - Run a one-time query and exit")
	fmt.Println("  -v      - Show version information")
	fmt.Println("  -debug  - Enable debug mode to log conversation data")

	fmt.Println("\nINTERACTIVE COMMANDS:")
	fmt.Println("  /clear  - Clear conversation history")
	fmt.Println("  /exit   - Exit the program")
	fmt.Println("  /quit   - Exit the program")
	fmt.Println("  /help   - Show help information")
}
