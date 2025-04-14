package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/pederhe/nca/internal/core"
	"github.com/pederhe/nca/internal/services/mcp"
	"github.com/pederhe/nca/pkg/api"
	"github.com/pederhe/nca/pkg/api/types"
	"github.com/pederhe/nca/pkg/config"
	"github.com/pederhe/nca/pkg/log"
	"github.com/pederhe/nca/pkg/utils"
)

// Version information, injected by compiler
var (
	Version    = "dev"
	BuildTime  = "unknown"
	CommitHash = "unknown"
)

// Add a global variable to cancel API requests
var (
	currentRequestCancel context.CancelFunc
	// Flag indicating whether an API request is being processed
	isProcessingAPIRequest bool
)

// Global variables for checkpoints
var (
	checkpointManager *core.CheckpointManager
)

// Mode selection: Agent or Ask
var (
	// true for Agent mode, false for Ask mode
	isAgentMode = true
	// number of times the conversation has been truncated
	conversationTruncatedCount = 0
)

func main() {
	// Initialize checkpoint manager
	checkpointManager = core.NewCheckpointManager()

	// Load checkpoints from file
	if err := checkpointManager.LoadCheckpoints(); err != nil {
		fmt.Printf("Warning: Failed to load checkpoints: %s\n", err)
	}

	// Initialize MCP hub
	mcpHub := mcp.GetMcpHub()
	defer mcpHub.Dispose()

	// No longer initialize signal handling here, let the readline library handle signals

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
		log.EnableDebugMode()
		defer log.CloseDebugLog()
		log.LogDebug("Program started with debug mode enabled\n")
	}

	args := flag.Args()

	// Process command line arguments
	if len(args) > 0 {
		switch args[0] {
		case "config":
			// Handle configuration settings command
			log.LogDebug(fmt.Sprintf("Config command: %v\n", args))
			handleConfigCommand(args[1:])
			return
		case "commit":
			// Handle git commit operation command
			log.LogDebug("Commit command detected\n")
			runREPL("commit all current changes, and summarize the changes")
			return
		case "help":
			// Display help information
			log.LogDebug("Help command detected\n")
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
		log.LogDebug("Detected pipe input\n")
		// Read from pipe
		reader := bufio.NewReader(os.Stdin)
		content, err := io.ReadAll(reader)
		if err != nil {
			fmt.Println("Error reading from pipe:", err)
			log.LogDebug(fmt.Sprintf("Error reading from pipe: %s\n", err))
			return
		}

		log.LogDebug(fmt.Sprintf("Pipe input content length: %d bytes\n", len(content)))

		// Combine pipe content with initial prompt if any
		if initialPrompt == "" {
			initialPrompt = string(content)
		} else {
			initialPrompt = initialPrompt + "\n\n" + string(content)
		}

		// When pipe input is detected, automatically run in one-time query mode
		if initialPrompt == "" {
			fmt.Println("Error: Empty pipe input")
			log.LogDebug("Error: Empty pipe input\n")
			return
		}
		log.LogDebug(fmt.Sprintf("Running one-time query mode with pipe input: %s\n", initialPrompt))
		runOneOffQuery(initialPrompt)
		return
	}

	// Run REPL or one-off query (only reached if no pipe input)
	if *promptFlag {
		if initialPrompt == "" {
			fmt.Println("Error: No prompt provided for one-time query")
			log.LogDebug("Error: No prompt provided for one-time query\n")
			return
		}
		log.LogDebug(fmt.Sprintf("One-time query mode with prompt: %s\n", initialPrompt))
		runOneOffQuery(initialPrompt)
	} else {
		log.LogDebug("Starting interactive REPL mode\n")
		if initialPrompt != "" {
			log.LogDebug(fmt.Sprintf("With initial prompt: %s\n", initialPrompt))
		}
		runREPL(initialPrompt)
	}
}

func getEnvironmentDetails() string {
	details := "\n# Current Mode\n"
	if isAgentMode {
		details += "AGENT MODE\n"
	} else {
		details += "ASK MODE\n"
	}

	// Get system language preference
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = "en_US.UTF-8" // Default to English if not set
	}
	lang = getLanguageCode(lang)
	details += fmt.Sprintf("\n# Preferred Language\nSpeak in %s\n", lang)

	return fmt.Sprintf("\n\n<environment_details>\n%s\n</environment_details>", details)
}

func getLanguageCode(lang string) string {
	if strings.Contains(lang, "zh") {
		return "中文"
	}
	return "English"
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

	// Check if there are any arguments left after removing the --global flag
	if len(cmdArgs) == 0 {
		fmt.Println("Usage: nca config [set|unset|list] [--global] [key] [value]")
		return
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
	// Track truncation state
	var currentDeletedRange [2]int

	// Log REPL start in debug mode
	log.LogDebug("Starting REPL session\n")
	if initialPrompt != "" {
		log.LogDebug(fmt.Sprintf("Initial prompt: %s\n", initialPrompt))
	}

	// If there's an initial prompt, handle it first
	if initialPrompt != "" {
		handlePrompt(initialPrompt, &conversation, &currentDeletedRange)
	} else {
		fmt.Printf("NCA %s (%s,%s)\n", Version, BuildTime, CommitHash)
		fmt.Println("Press Ctrl+A to toggle between [Agent] and [Ask] mode")
		if log.IsDebugMode() {
			fmt.Print(utils.ColoredText("Debug mode enabled. Logs saved to: "+log.GetDebugLogPath()+"\n", utils.ColorYellow))
		}
	}

	// Create custom completer for commands
	completer := readline.NewPrefixCompleter(
		readline.PcItem("/clear"),
		readline.PcItem("/checkpoint",
			readline.PcItem("list"),
			readline.PcItem("restore"),
			readline.PcItem("redo"),
		),
		readline.PcItem("/config",
			readline.PcItem("set"),
			readline.PcItem("unset"),
			readline.PcItem("list"),
			readline.PcItem("--global"),
		),
		readline.PcItem("/mcp",
			readline.PcItem("list"),
			readline.PcItem("reload"),
		),
		readline.PcItem("/help"),
		readline.PcItem("/exit"),
	)

	// Create a custom interrupt handler
	interruptHandler := func() {
		if isProcessingAPIRequest && currentRequestCancel != nil {
			// If an API request is in progress, cancel it
			log.LogDebug("Cancelling current API request due to interrupt\n")
			currentRequestCancel()
			fmt.Println("\nAPI request cancelled")
		}
	}

	// Get the appropriate prompt prefix based on current mode
	getPromptPrefix := func() string {
		if isAgentMode {
			return "[Agent]>>> "
		}
		return "[Ask]>>> "
	}

	// Initialize readline configuration
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            utils.ColoredText(getPromptPrefix(), utils.ColorPurple),
		HistoryFile:       os.Getenv("HOME") + "/.nca_history",
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true, // Case-insensitive history search
		AutoComplete:      completer,
	})
	if err != nil {
		fmt.Println("Error initializing readline:", err)
		log.LogDebug(fmt.Sprintf("Error initializing readline: %s\n", err))
		return
	}
	defer rl.Close()

	// Set up Ctrl+A key handling for mode switching
	// When user presses Ctrl+A, switch between Agent and Ask modes
	oldHandler := rl.Config.FuncFilterInputRune
	rl.Config.FuncFilterInputRune = func(r rune) (rune, bool) {
		if r == 1 { // Ctrl+A key (ASCII 1)
			// Only allow mode switching if not processing an API request
			if !isProcessingAPIRequest {
				isAgentMode = !isAgentMode
				newPrompt := getPromptPrefix()
				rl.SetPrompt(utils.ColoredText(newPrompt, utils.ColorPurple))
				rl.Refresh()
				log.LogDebug(fmt.Sprintf("Mode switched to: %s\n", newPrompt))
			}
			return r, true // rl.Close will block the program if we return false here
		}
		// Pass to original handler if set
		if oldHandler != nil {
			return oldHandler(r)
		}
		return r, true
	}

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Start signal handling goroutine
	go func() {
		for range signalChan {
			interruptHandler()
		}
	}()

	// Accumulate multi-line input
	multilineBuffer := ""
	clipboardMode := false

	for {
		// Read input using readline
		input, err := rl.Readline()
		if err != nil {
			// Handle Ctrl+C or Ctrl+D
			if err == readline.ErrInterrupt {
				fmt.Println("Interrupted")
				log.LogDebug("User interrupted input with Ctrl+C\n")
				continue
			} else if err == io.EOF {
				log.LogDebug("User exited with Ctrl+D\n")
				handleExit("with Ctrl+D")
				break
			}
			fmt.Println("Error reading input:", err)
			log.LogDebug(fmt.Sprintf("Error reading input: %s\n", err))
			continue
		}

		// If it's an empty line and there's no accumulated input, continue reading
		if strings.TrimSpace(input) == "" && multilineBuffer == "" {
			continue
		}

		// Check if we need to enter or continue clipboard mode
		if !clipboardMode {
			isPrefix, content, err := utils.IsClipboardPrefix(input)
			// If the input is exactly the same as the clipboard content, don't enter clipboard mode, as this means only one line of data was pasted
			if err == nil && isPrefix && strings.TrimSpace(input) != strings.TrimSpace(content) {
				clipboardMode = true
				multilineBuffer = input
				rl.SetPrompt(utils.ColoredText("... ", utils.ColorPurple))
				continue
			}
		} else {
			// In clipboard mode, there are multiple lines of data
			// If it's an empty line, exit clipboard mode
			if strings.TrimSpace(input) == "" {
				clipboardMode = false
				finalInput := multilineBuffer
				multilineBuffer = ""
				rl.SetPrompt(utils.ColoredText(getPromptPrefix(), utils.ColorPurple))
				if handleInput(finalInput, &conversation, &currentDeletedRange) {
					break
				}
				continue
			}

			// Continue adding input to the buffer
			multilineBuffer += "\n" + input
			continue
		}

		// Process regular multi-line input (non-clipboard mode)
		if multilineBuffer != "" {
			// Empty line with accumulated multiline input means end of input
			if strings.TrimSpace(input) == "" {
				finalInput := multilineBuffer
				multilineBuffer = ""                                                  // Reset the buffer
				rl.SetPrompt(utils.ColoredText(getPromptPrefix(), utils.ColorPurple)) // Reset prompt to primary prompt

				// Handle the complete multiline input
				if handleInput(finalInput, &conversation, &currentDeletedRange) {
					break
				}
				continue
			}

			// No backslash needed for continuation lines, add directly
			multilineBuffer += "\n" + input
			continue
		}

		// Check if input ends with backslash to start multi-line mode
		if strings.HasSuffix(strings.TrimSpace(input), "\\") {
			// Start accumulating multi-line input
			// Remove the trailing backslash for the first line
			trimmedInput := strings.TrimSpace(input)
			multilineBuffer = trimmedInput[:len(trimmedInput)-1] // Remove the trailing backslash
			// Set secondary prompt for continuation lines
			rl.SetPrompt(utils.ColoredText("... ", utils.ColorPurple))
			continue
		}

		lastLen := len(conversation)
		// Handle single line input
		if handleInput(input, &conversation, &currentDeletedRange) {
			break
		}
		if len(conversation) < lastLen {
			conversationTruncatedCount++
		}
		if conversationTruncatedCount >= 3 {
			// TODO use the previous conversation summary as the initial conversation for the new session
			fmt.Println(utils.ColoredText("Use /clear to start a new conversation for better results.", utils.ColorCyan))
		}
	}

	// Clean up signal handling
	signal.Stop(signalChan)
	close(signalChan)
}

func handleExit(exitReason string) {
	fmt.Println("Exiting")
	log.LogDebug(fmt.Sprintf("User exited: %s\n", exitReason))

	// Save checkpoints before exit
	if err := checkpointManager.SaveCheckpoints(); err != nil {
		fmt.Printf("Warning: Failed to save checkpoints: %s\n", err)
		log.LogDebug(fmt.Sprintf("Failed to save checkpoints: %s\n", err))
	}
}

// Add a general function to handle input
func handleInput(input string, conversation *[]map[string]string, currentDeletedRange *[2]int) bool {
	// If it's a command
	if strings.HasPrefix(input, "/") {
		log.LogDebug(fmt.Sprintf("Slash command: %s\n", input))
		if input == "/exit" {
			handleExit("with /exit command")
			return true // Indicates need to exit
		}
		handleSlashCommand(input, conversation, currentDeletedRange)
	} else {
		// Normal input processing
		handlePrompt(input, conversation, currentDeletedRange)
	}
	return false // Indicates no need to exit
}

// Run one-off query
func runOneOffQuery(prompt string) {
	conversation := []map[string]string{}
	var currentDeletedRange [2]int
	log.LogDebug("Running one-off query mode\n")
	log.LogDebug(fmt.Sprintf("Query: %s\n", prompt))

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Create a custom interrupt handler
	interruptHandler := func() {
		if isProcessingAPIRequest && currentRequestCancel != nil {
			// If an API request is in progress, cancel it
			log.LogDebug("Cancelling current API request due to interrupt\n")
			currentRequestCancel()
			fmt.Println("\nAPI request cancelled")
		}
	}

	// Start signal handling goroutine
	go func() {
		for range signalChan {
			interruptHandler()
		}
	}()

	handlePrompt(prompt, &conversation, &currentDeletedRange)

	// Clean up signal handling
	signal.Stop(signalChan)
	close(signalChan)

	log.LogDebug("One-off query completed\n")
}

// Handle user input prompt
func handlePrompt(prompt string, conversation *[]map[string]string, currentDeletedRange *[2]int) {
	// Create a checkpoint at the beginning of each prompt handling
	checkpointManager.CreateCheckpoint(prompt)

	// Check if the prompt contains files or URLs to be processed
	// This helps users understand that their files or URLs are being processed
	if utils.HasBackticks(prompt) {
		fmt.Print("\nProcessing resources in prompt... ")
		log.LogDebug("Detected backticks in prompt, processing resources\n")

		newPrompt, err := utils.ProcessPrompt(prompt)
		if err != nil {
			fmt.Println(utils.ColoredText("Error processing prompt: "+err.Error(), utils.ColorRed))
			return
		}
		prompt = newPrompt
		fmt.Println("Done")
		fmt.Println()
	}

	// Add user message to conversation history
	*conversation = append(*conversation, map[string]string{
		"role":    "user",
		"content": prompt + getEnvironmentDetails(),
	})

	// Log user input in debug mode
	log.LogDebug(fmt.Sprintf("USER INPUT (Mode: %s): %s\n",
		map[bool]string{true: "Agent", false: "Ask"}[isAgentMode], prompt))

	// Count of consecutive responses without tool use
	noToolUseCount := 0

	// Message count limit
	maxMessagesPerTask := 25

	// Multi-step task processing loop
	for {
		// Check if message count has reached the limit
		if maxMessagesPerTask <= 0 {
			limitMessage := "Maximum of 25 requests per task reached, system has automatically exited"
			fmt.Println(utils.ColoredText(limitMessage, utils.ColorYellow))
			log.LogDebug(fmt.Sprintf("MESSAGE LIMIT REACHED: %s\n", limitMessage))
			break
		}

		// Create API client
		client, err := api.NewClient()
		if err != nil {
			fmt.Println("Error: Failed to create API client:", err)
			break
		}
		// Call API
		response, err := callAPI(client, *conversation)
		if err != nil {
			fmt.Println("Error calling API:", err)
			log.LogDebug(fmt.Sprintf("API ERROR: %s\n", err))

			// Add error message to conversation history
			*conversation = append(*conversation, map[string]string{
				"role":    "assistant",
				"content": err.Error(),
			})

			break
		}
		debugPrintUsage(response.Usage)
		maxMessagesPerTask--

		// if the finish_reason is "length", it means the context length is insufficient, so we need to cut off the previous conversation
		if response.FinishReason == "length" {
			newRange := core.GetNextTruncationRange(*conversation, *currentDeletedRange, "quarter")
			// If we can't truncate any more messages, exit
			if newRange[1] <= newRange[0] {
				fmt.Println(utils.ColoredText("Context length exceeded and cannot be truncated further. Please use /clear to start a new conversation.", utils.ColorRed))
				break
			}

			// Update current deleted range
			*currentDeletedRange = newRange

			// Create new conversation slice with truncated messages
			// Keep messages before the truncation range and after the truncation range
			*conversation = append((*conversation)[:newRange[0]], (*conversation)[newRange[1]+1:]...)

			// Log truncation in debug mode
			log.LogDebug(fmt.Sprintf("Context truncated. Removed messages %d-%d\n", newRange[0], newRange[1]))

			// Continue with truncated conversation
			continue
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
			log.LogDebug(fmt.Sprintf("TOOL USE: %v\n", toolUse))

			result := handleToolUse(toolUse)
			if toolName == "replace_in_file" {
				lines := strings.SplitN(result, "\n", 2)
				if len(lines) == 2 {
					result = lines[0]
					fmt.Println(lines[1])
				}
			}

			// Log tool result in debug mode
			log.LogDebug(fmt.Sprintf("TOOL RESULT: %s\n", result))

			// Get tool name (already extracted above)
			// Check if it's the task completion tool
			if toolName == "attempt_completion" {
				fmt.Println(utils.ColoredText(result, utils.ColorYellow))
				// Task completed, exit loop
				break
			}
			if toolName == "ask_mode_response" || toolName == "ask_followup_question" {
				// Task completed, exit loop
				break
			}

			// Format tool description based on tool type
			toolDesc := formatToolDescription(toolUse)

			// Add tool result to conversation history with description
			// some models return multiple tools, so we need to tell them to only use one tool per message
			toolResultContent := fmt.Sprintf("%s Result:\n%s", toolDesc, result)
			if _, exists := toolUse["has_multiple_tools"]; exists {
				toolResultContent += "\n\nOnly one tool may be used per message. You must assess the first tool's result before proceeding to use the next tool."
			}
			*conversation = append(*conversation, map[string]string{
				"role":    "user",
				"content": toolResultContent,
			})

			// Continue loop, process next step
		} else {
			log.LogDebug(fmt.Sprintf("ERROR: No tool use response, content: %s\n", response.Content))
			// Increment counter for responses without tool use
			noToolUseCount++

			// Check if exceeded 3 attempts without tool use
			if noToolUseCount >= 3 {
				errorMessage := "[FATAL ERROR] You failed to use a tool after 3 attempts. Exiting task."
				log.LogDebug(fmt.Sprintf("ERROR: %s\n", errorMessage))
				*conversation = append(*conversation, map[string]string{
					"role":    "user",
					"content": errorMessage,
				})
				fmt.Println(utils.ColoredText("System error. You can use /clear to start a new conversation.", utils.ColorRed))
				break
			}

			// No tool use request, add error message to conversation history
			errorMessage := fmt.Sprintf("[ERROR] You did not use a tool in your previous response! Please retry with a tool use. (Attempt %d/3)", noToolUseCount)
			log.LogDebug(fmt.Sprintf("ERROR: %s\n", errorMessage))
			*conversation = append(*conversation, map[string]string{
				"role":    "user",
				"content": errorMessage,
			})
			fmt.Println(utils.ColoredText("No available tools found", utils.ColorRed))
			// Don't exit loop, continue requesting AI to use a tool
		}
		// Update the context messages
		core.UpdateContextMessages(client.GetModelInfo(), conversation, currentDeletedRange, response.Usage)
	}
}

// Format tool description based on tool type and parameters
func formatToolDescription(toolUse map[string]interface{}) string {
	toolName, _ := toolUse["tool"].(string)

	switch toolName {
	case "attempt_completion":
		return "[attempt_completion]"

	case "ask_mode_response":
		return "[ask_mode_response]"

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

	case "use_mcp_tool":
		serverName, _ := toolUse["server_name"].(string)
		toolNameParam, _ := toolUse["tool_name"].(string)
		return fmt.Sprintf("[%s for server '%s', tool '%s']", toolName, serverName, toolNameParam)

	case "access_mcp_resource":
		serverName, _ := toolUse["server_name"].(string)
		uri, _ := toolUse["uri"].(string)
		return fmt.Sprintf("[%s for server '%s', uri '%s']", toolName, serverName, uri)

	case "find_files":
		return "[find_files]"

	default:
		return fmt.Sprintf("[%s]", toolName)
	}
}

// Handle slash command
func handleSlashCommand(cmd string, conversation *[]map[string]string, currentDeletedRange *[2]int) {
	// Handle /checkpoint command
	if strings.HasPrefix(cmd, "/checkpoint") {
		args := strings.Fields(cmd)
		// Remove the "/checkpoint" prefix and pass the remaining arguments
		var cmdArgs []string
		if len(args) > 1 {
			cmdArgs = args[1:]
		}
		result := checkpointManager.HandleCheckpointCommand(cmdArgs)
		fmt.Println(result)
		log.LogDebug(fmt.Sprintf("Checkpoint command executed: %s\nResult: %s\n", cmd, result))
		return
	}

	// Handle /config command, format: "/config [set|unset|list] [--global] [key] [value]"
	if strings.HasPrefix(cmd, "/config") {
		args := strings.Fields(cmd)
		if len(args) > 1 {
			// Remove the "/config" prefix and pass the remaining arguments to handleConfigCommand
			handleConfigCommand(args[1:])
		} else {
			// If there's only "/config" without other arguments, show usage
			fmt.Println("Usage: /config [set|unset|list] [--global] [key] [value]")
		}
		log.LogDebug(fmt.Sprintf("Config command executed in interactive mode: %s\n", cmd))
		return
	}

	// Handle /mcp command, format: "/mcp [list|reload]"
	if strings.HasPrefix(cmd, "/mcp") {
		args := strings.Fields(cmd)
		if len(args) > 1 {
			switch args[1] {
			case "list":
				// Get MCPHub and show server connections
				mcp.GetMcpHub().PrintConnections()
			case "reload":
				// Reload MCP servers
				mcp.GetMcpHub().ReloadServers()
				fmt.Println(utils.ColoredText("MCP servers reloaded", utils.ColorGreen))
				// Show updated server connections
				mcp.GetMcpHub().PrintConnections()
				log.LogDebug("MCP reload command executed\n")
			default:
				fmt.Println("Unknown MCP command. Available commands: list, reload")
			}
		} else {
			// If there's only "/mcp" without other arguments, show usage
			fmt.Println("Usage: /mcp [list|reload]")
		}
		return
	}

	switch cmd {
	case "/clear":
		*conversation = []map[string]string{}
		*currentDeletedRange = [2]int{0, 0}
		conversationTruncatedCount = 0
		fmt.Println("Conversation history cleared")
		fmt.Println(utils.ColoredText("----------------New Chat----------------", utils.ColorBlue))
		log.LogDebug("Conversation history cleared by user\n")
	case "/help":
		fmt.Println("\nINTERACTIVE COMMANDS:")
		fmt.Println("  /clear      - Clear conversation history")
		fmt.Println("  /config     - Manage configuration settings")
		fmt.Println("               Usage: /config [set|unset|list] [--global] [key] [value]")
		fmt.Println("  /checkpoint - Manage checkpoints")
		fmt.Println("               Usage: /checkpoint [list|restore|redo] [checkpoint_id]")
		fmt.Println("  /mcp        - Manage MCP server connections")
		fmt.Println("               Usage: /mcp [list|reload]")
		fmt.Println("  /exit       - Exit the program")
		fmt.Println("  /help       - Show help information")
		log.LogDebug("Help information displayed\n")
	case "/exit":
		// These are handled in the runREPL function
		// Nothing to do here
	default:
		fmt.Println("Unknown command. Enter /help for help")
		log.LogDebug(fmt.Sprintf("Unknown command attempted: %s\n", cmd))
	}
}

// API response structure
type APIResponse struct {
	ReasoningContent string       `json:"reasoning_content"`
	Content          string       `json:"content"`
	Usage            *types.Usage `json:"usage"`
	FinishReason     string       `json:"finish_reason"`
}

// Call AI API
func callAPI(client *api.Client, conversation []map[string]string) (APIResponse, error) {
	// Set flag indicating an API request is being processed
	isProcessingAPIRequest = true

	// Ensure the flag is cleared when the function returns
	defer func() {
		isProcessingAPIRequest = false
	}()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	// Save the cancel function to the global variable so it can be called from elsewhere
	currentRequestCancel = cancel
	// Ensure the cancel function is cleared when the function returns
	defer func() {
		currentRequestCancel = nil
	}()

	// Build system prompt
	systemPrompt, err := core.BuildSystemPrompt()
	if err != nil {
		log.LogDebug(fmt.Sprintf("ERROR building system prompt: %s\n", err))
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
	log.LogDebug("API REQUEST PAYLOAD:\n")
	for i, msg := range messages {
		// Truncate system message for brevity in logs
		content := msg.Content
		if msg.Role == "system" && len(content) > 100 {
			content = content[:100] + "... [truncated]"
		}
		log.LogDebug(fmt.Sprintf("  [%d] %s: %s\n", i, msg.Role, content))
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

	// Create a channel to receive API response results
	resultCh := make(chan struct {
		reasoningContent string
		content          string
		usage            *types.Usage
		finishReason     string
		err              error
	}, 1)

	// Call the API in a goroutine so it can be cancelled via the context
	go func() {
		// Define callback function for streaming
		callback := func(reasoningChunk string, chunk string, isDone bool) {
			// Check if the context has been cancelled
			select {
			case <-ctx.Done():
				return // If the context has been cancelled, don't process the callback
			default:
				// Continue normal processing
			}

			if reasoningChunk != "" {
				if !startReasoning {
					startReasoning = true
					fmt.Println(utils.ColoredText("Reasoning:", utils.ColorBlue))
				}
				// Stop loading animation when first reasoning chunk is received
				if len(reasoningChunk) > 0 && !animationStopped {
					stopLoading <- true
					<-animationDone // Wait for animation to actually stop
					animationStopped = true
				}
				fmt.Print(reasoningChunk)
			} else if chunk != "" {
				if startReasoning {
					fmt.Println(utils.ColoredText("\n----------------------------", utils.ColorBlue))
					startReasoning = false
				}
				// Filter and print the chunk
				filtered := filter.ProcessChunk(chunk)
				// Stop loading animation when first available chunk is received
				if len(filtered) > 0 && !animationStopped {
					stopLoading <- true
					<-animationDone // Wait for animation to actually stop
					animationStopped = true
				}
				fmt.Print(filtered)
			}
		}

		// Call API with streaming, passing the context
		response, err := client.ChatStream(ctx, messages, callback)
		if err != nil {
			resultCh <- struct {
				reasoningContent string
				content          string
				usage            *types.Usage
				finishReason     string
				err              error
			}{"", "", nil, "", err}
			return
		}

		// Send the result to the channel
		resultCh <- struct {
			reasoningContent string
			content          string
			usage            *types.Usage
			finishReason     string
			err              error
		}{response.ReasoningContent, response.Content, response.Usage, response.FinishReason, nil}
	}()

	// Wait for the API call to complete or the context to be cancelled
	var reasoningContent, content string
	var usage *types.Usage
	var finishReason string
	var apiErr error

	select {
	case <-ctx.Done():
		// Context was cancelled (user pressed Ctrl+C)
		log.LogDebug("API request cancelled by user\n")
		apiErr = fmt.Errorf("request cancelled by user")
	case result := <-resultCh:
		// API call completed
		reasoningContent = result.reasoningContent
		content = result.content
		usage = result.usage
		finishReason = result.finishReason
		apiErr = result.err
	}

	//fmt.Println() // Add newline after streaming completes

	// Log raw response in debug mode
	if apiErr == nil {
		log.LogDebug(fmt.Sprintf("RAW API RESPONSE STREAM:\n%s\n%s\n%s\n",
			reasoningContent, "--------------------------------", content))
	} else {
		log.LogDebug(fmt.Sprintf("API REQUEST CANCELLED OR ERROR: %s\n", apiErr))
	}

	// Ensure loading animation is stopped
	if !animationStopped {
		stopLoading <- true
		<-animationDone
	}

	if apiErr != nil {
		log.LogDebug(fmt.Sprintf("API STREAM ERROR: %s\n", apiErr))
		return APIResponse{}, fmt.Errorf("API call error: %s", apiErr)
	}

	return APIResponse{
		ReasoningContent: reasoningContent,
		Content:          content,
		Usage:            usage,
		FinishReason:     finishReason,
	}, nil
}

// Display loading animation
func showLoadingAnimation(stop chan bool, done chan bool) {
	// If output is to a pipe, don't show animation
	if utils.IsOutputPiped() {
		// Immediately return completion signal, but keep channel open to avoid blocking
		go func() {
			<-stop
			done <- true
		}()
		return
	}

	// Loading animation characters
	spinChars := []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"}
	i := 0

	// Clear current line and display initial message
	fmt.Print("\r")

	for {
		select {
		case <-stop:
			// Clear animation line to ensure it doesn't affect subsequent output
			fmt.Print("\r                        \r")
			done <- true // Notify that animation has stopped
			return
		default:
			// Display spinning animation
			fmt.Printf("\r%s", spinChars[i])
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

	// If this is a command that might delete files, track it via execute_command
	if toolName == "execute_command" {
		// Get the command
		command, cmdOk := toolUse["command"].(string)
		if cmdOk {
			// Check if the command is likely to delete files (rm, del, etc.)
			commandLower := strings.ToLower(command)
			if strings.Contains(commandLower, "rm ") || strings.Contains(commandLower, "del ") ||
				strings.Contains(commandLower, "remove ") || strings.Contains(commandLower, "rmdir ") {
				// Extract potential file paths from the command
				parts := strings.Fields(command)
				for i := 1; i < len(parts); i++ {
					path := parts[i]
					// Skip flags
					if strings.HasPrefix(path, "-") {
						continue
					}

					// Check if file exists before executing
					if fileInfo, err := os.Stat(path); err == nil && !fileInfo.IsDir() {
						// Read file content for potential restoration
						if content, err := os.ReadFile(path); err == nil {
							checkpointManager.RecordFileOperation("delete", path, string(content), "")
						}
					}
				}
			}
		}
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

	// Execute the appropriate tool function
	var result string
	switch toolName {
	case "execute_command":
		result = core.ExecuteCommand(toolUse)
	case "read_file":
		result = core.ReadFile(toolUse)
	case "write_to_file":
		// Get the file path and content
		path, pathOk := toolUse["path"].(string)
		content, contentOk := toolUse["content"].(string)

		if pathOk && contentOk {
			// Check if file exists before writing
			oldContent := ""
			if fileContent, err := os.ReadFile(path); err == nil {
				oldContent = string(fileContent)
			}

			// Record the file operation
			if oldContent == "" {
				// New file
				checkpointManager.RecordFileOperation("write", path, content, "")
			} else {
				// Existing file
				checkpointManager.RecordFileOperation("replace", path, content, oldContent)
			}
		}

		result = core.WriteToFile(toolUse)
	case "replace_in_file":
		// Get the file path and diff
		path, pathOk := toolUse["path"].(string)
		_, diffOk := toolUse["diff"].(string)

		if pathOk && diffOk {
			// Get current content for undo
			oldContent := ""
			if fileContent, err := os.ReadFile(path); err == nil {
				oldContent = string(fileContent)
			}

			// Record the operation, but set the final content after execution
			recordOperation := func(newContent string) {
				checkpointManager.RecordFileOperation("replace", path, newContent, oldContent)
			}

			// Get replacement result
			result = core.ReplaceInFile(toolUse)

			// Extract the new content by reading the file again
			if newContent, err := os.ReadFile(path); err == nil {
				recordOperation(string(newContent))
			}
		} else {
			result = core.ReplaceInFile(toolUse)
		}
	case "search_files":
		result = core.SearchFiles(toolUse)
	case "list_files":
		result = core.ListFiles(toolUse)
	case "list_code_definition_names":
		result = core.ListCodeDefinitionNames(toolUse)
	case "ask_followup_question":
		result = core.FollowupQuestion(toolUse)
	case "ask_mode_response":
		result = core.AskModeResponse(toolUse)
	case "git_commit":
		result = core.GitCommit(toolUse)
	case "fetch_web_content":
		result = core.FetchWebContent(toolUse)
	case "find_files":
		result = core.FindFiles(toolUse)
	case "use_mcp_tool":
		result = core.UseMcpTool(toolUse)
	case "access_mcp_resource":
		result = core.AccessMcpResource(toolUse)
	default:
		result = fmt.Sprintf("Error: Unknown tool '%s'", toolName)
	}

	return result
}

func debugPrintUsage(usage *types.Usage) {
	if !log.IsDebugMode() {
		return
	}
	usageStr := fmt.Sprintf("\nPrompt tokens: %d, Completion tokens: %d, Total tokens: %d\n",
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	fmt.Print(usageStr)

	log.LogDebug(usageStr)
}

// displayHelp shows all available commands and options
func displayHelp() {
	fmt.Println("NCA - Nano Code Agent")
	fmt.Printf("Version: %s, Build time: %s, Commit hash: %s\n\n", Version, BuildTime, CommitHash)

	fmt.Println("USAGE:")
	fmt.Println("  nca [options] [prompt]")
	fmt.Println("  nca [command]")

	fmt.Println("\nPROMPT FEATURES:")
	fmt.Println("  File Reading   - Include file content by wrapping the path in backticks: `path/to/file.txt`")
	fmt.Println("  Web Content    - Include web content by wrapping the URL in backticks: `https://example.com`")
	fmt.Println("  Multiple Files - You can include multiple files or URLs in the same prompt")
	fmt.Println("  Size Limits    - Files are limited to 64KB, web content is filtered to extract text")

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
	fmt.Println("  /clear      - Clear conversation history")
	fmt.Println("  /config     - Manage configuration settings")
	fmt.Println("               Usage: /config [set|unset|list] [--global] [key] [value]")
	fmt.Println("  /checkpoint - Manage checkpoints")
	fmt.Println("               Usage: /checkpoint [list|restore|redo] [checkpoint_id]")
	fmt.Println("  /mcp        - Manage MCP server connections")
	fmt.Println("               Usage: /mcp [list|reload]")
	fmt.Println("  /exit       - Exit the program")
	fmt.Println("  /help       - Show help information")
}
