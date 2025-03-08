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
	"github.com/pederhe/nca/config"
	"github.com/pederhe/nca/core"
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

	// Handle config command
	if len(args) > 0 && args[0] == "config" {
		logDebug(fmt.Sprintf("Config command: %v\n", args))
		handleConfigCommand(args[1:])
		return
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
			fmt.Printf(core.ColorYellow+"Debug mode enabled. Logs saved to: %s\n"+core.ColorReset, debugLogPath)
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
		Prompt:            core.ColorPurple + ">>> " + core.ColorReset,
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
			logDebug(fmt.Sprintf("TOOL USE: %s\n", toolName))

			result := handleToolUse(toolUse)

			// Log tool result in debug mode
			logDebug(fmt.Sprintf("TOOL RESULT: %s\n", result))

			// Format tool description based on tool type
			toolDesc := formatToolDescription(toolUse)

			// Add tool result to conversation history with description
			toolResultContent := fmt.Sprintf("%s Result:\n%s", toolDesc, result)
			*conversation = append(*conversation, map[string]string{
				"role":    "user",
				"content": toolResultContent,
			})

			// Get tool name (already extracted above)
			// Check if it's the task completion tool
			if toolName == "attempt_completion" {
				fmt.Println(core.ColorYellow + result + core.ColorReset)
				// Task completed, exit loop
				break
			}
			if toolName == "plan_mode_response" || toolName == "ask_followup_question" {
				// Task completed, exit loop
				break
			}

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
		fmt.Println("Available commands:")
		fmt.Println("/clear - Clear conversation history")
		fmt.Println("/exit, /quit - Exit the program")
		fmt.Println("/help - Show help information")
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
	Content string `json:"content"`
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
	client := api.NewClient()

	// Start dynamic loading animation
	stopLoading := make(chan bool, 1)   // Use buffered channel to prevent blocking
	animationDone := make(chan bool, 1) // Channel to confirm animation has stopped
	go showLoadingAnimation(stopLoading, animationDone)

	// Create a filter for XML tags
	filter := newXMLTagFilter()

	// Flag to track if animation has been stopped
	var animationStopped bool = false

	// Define callback function for streaming
	callback := func(chunk string, isDone bool) {
		// Stop loading animation when first response chunk is received
		if len(chunk) > 0 && !animationStopped {
			stopLoading <- true
			<-animationDone // Wait for animation to actually stop
			animationStopped = true
		}

		// Filter and print the chunk
		filtered := filter.processChunk(chunk)
		fmt.Print(filtered)
	}

	// Call API with streaming
	content, err := client.ChatStream(messages, callback)
	fmt.Println() // Add newline after streaming completes

	// Log raw response in debug mode
	logDebug(fmt.Sprintf("RAW API RESPONSE STREAM:\n%s\n", content))

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
	cleanedContent := core.RemoveThinkingTags(content)

	return APIResponse{
		Content: cleanedContent,
	}, nil
}

// Display loading animation
func showLoadingAnimation(stop chan bool, done chan bool) {
	// Loading animation characters
	spinChars := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
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
			fmt.Printf("\r\033[K%s Generating... ", spinChars[i])
			i = (i + 1) % len(spinChars)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// XMLTagFilter filters out XML tool tags from text
type XMLTagFilter struct {
	buffer        strings.Builder
	tagStack      []string
	currentTag    strings.Builder
	collectingTag bool
	inToolTag     bool
	inSubTag      bool
	currentSubTag string
	inDiffTag     bool            // Whether inside a diff tag
	inContentTag  bool            // Whether inside a content tag
	pendingBuffer strings.Builder // Buffer for storing potential tag start sequences
}

// Create a new XML tag filter
func newXMLTagFilter() *XMLTagFilter {
	return &XMLTagFilter{
		buffer:        strings.Builder{},
		tagStack:      []string{},
		currentTag:    strings.Builder{},
		collectingTag: false,
		inToolTag:     false,
		inSubTag:      false,
		currentSubTag: "",
		inDiffTag:     false,
		inContentTag:  false,
		pendingBuffer: strings.Builder{},
	}
}

// Process a chunk of text and filter out XML tool tags
func (f *XMLTagFilter) processChunk(chunk string) string {
	f.buffer.Reset()

	// If there are pending characters, prepend them to the chunk
	if f.pendingBuffer.Len() > 0 {
		chunk = f.pendingBuffer.String() + chunk
		f.pendingBuffer.Reset()
	}

	for i := 0; i < len(chunk); i++ {
		c := chunk[i]

		// Handle special tags (diff, content) that don't filter their content
		if processed, newIndex := f.handleSpecialTags(chunk, i); processed {
			i = newIndex
			continue
		}

		// Handle tag start
		if c == '<' {
			// Check for special tag openings
			if processed, newIndex := f.handleSpecialTagOpening(chunk, i); processed {
				i = newIndex
				continue
			}

			// Not a special tag, process normally
			f.collectingTag = true
			f.currentTag.Reset()
			continue
		}

		// Handle tag end
		if c == '>' && f.collectingTag {
			f.collectingTag = false
			tag := f.currentTag.String()

			// Process the tag
			f.processTag(tag)
			continue
		}

		// Collect tag name
		if f.collectingTag {
			f.currentTag.WriteByte(c)
			continue
		}

		// Output character if:
		// 1. Not in a tool tag, or
		// 2. In a sub-tag inside a tool tag (except hidden tags)
		if !f.inToolTag || (f.inSubTag && !isHiddenTag(f.currentSubTag)) {
			f.buffer.WriteByte(c)
		}
	}

	return f.buffer.String()
}

// Handle special tags like diff and content that don't filter their content
func (f *XMLTagFilter) handleSpecialTags(chunk string, index int) (bool, int) {
	// If inside a diff tag
	if f.inDiffTag {
		return f.handleDiffTagContent(chunk, index)
	}

	// If inside a content tag
	if f.inContentTag {
		return f.handleContentTagContent(chunk, index)
	}

	return false, index
}

// Handle content inside diff tags
func (f *XMLTagFilter) handleDiffTagContent(chunk string, index int) (bool, int) {
	c := chunk[index]

	// Check if this might be the start of a </diff> closing tag
	if c == '<' {
		// If not enough characters left to determine if it's a </diff> tag, store in pendingBuffer
		if index+6 >= len(chunk) {
			f.pendingBuffer.WriteString(chunk[index:])
			return true, len(chunk)
		}

		// Check if it's a </diff> closing tag
		if chunk[index:index+7] == "</diff>" {
			// Don't output the </diff> tag
			f.inDiffTag = false
			return true, index + 6
		}
	}

	// Inside diff tag, output character directly
	f.buffer.WriteByte(c)
	return true, index
}

// Handle content inside content tags
func (f *XMLTagFilter) handleContentTagContent(chunk string, index int) (bool, int) {
	c := chunk[index]

	// Check if this might be the start of a </content> closing tag
	if c == '<' {
		// If not enough characters left to determine if it's a </content> tag, store in pendingBuffer
		if index+9 >= len(chunk) {
			f.pendingBuffer.WriteString(chunk[index:])
			return true, len(chunk)
		}

		// Check if it's a </content> closing tag
		if chunk[index:index+10] == "</content>" {
			// Don't output the </content> tag
			f.inContentTag = false
			return true, index + 9
		}
	}

	// Inside content tag, output character directly
	f.buffer.WriteByte(c)
	return true, index
}

// Handle opening of special tags
func (f *XMLTagFilter) handleSpecialTagOpening(chunk string, index int) (bool, int) {
	// Check for <diff> tag
	if index+5 < len(chunk) && chunk[index:index+6] == "<diff>" {
		// Don't output the <diff> tag
		f.inDiffTag = true
		return true, index + 5
	}

	// Check for <content> tag
	if index+8 < len(chunk) && chunk[index:index+9] == "<content>" {
		// Don't output the <content> tag
		f.inContentTag = true
		return true, index + 8
	}

	// If not enough characters left to determine if it's a special tag, store in pendingBuffer
	if index+8 >= len(chunk) {
		f.pendingBuffer.WriteString(chunk[index:])
		return true, len(chunk) - 1
	}

	return false, index
}

// Process a tag (opening or closing)
func (f *XMLTagFilter) processTag(tag string) {
	// Check if it's a closing tag
	if strings.HasPrefix(tag, "/") {
		f.processClosingTag(tag[1:]) // Remove the leading '/'
	} else {
		f.processOpeningTag(tag)
	}
}

// Process a closing tag
func (f *XMLTagFilter) processClosingTag(tagName string) {
	// Check if we're closing a tag in our stack
	if len(f.tagStack) > 0 && f.tagStack[len(f.tagStack)-1] == tagName {
		// Pop the tag from stack
		f.tagStack = f.tagStack[:len(f.tagStack)-1]

		// If we're closing the root tool tag, exit tool tag mode
		if len(f.tagStack) == 0 && isToolTag(tagName) {
			f.inToolTag = false
			f.inSubTag = false
			f.currentSubTag = ""
		} else if f.inToolTag && len(f.tagStack) == 1 {
			// We're closing a sub-tag inside a tool tag

			// Reset color before closing the sub-tag
			f.buffer.WriteString(core.ColorReset)

			f.inSubTag = false
			f.currentSubTag = ""

			// Add a newline after the sub-tag content for better formatting
			f.buffer.WriteByte('\n')
		}
	}
}

// Process an opening tag
func (f *XMLTagFilter) processOpeningTag(tag string) {
	// If it's a root tool tag, enter tool tag mode but don't output the tag
	if len(f.tagStack) == 0 && isToolTag(tag) {
		f.inToolTag = true
	} else if f.inToolTag && len(f.tagStack) == 1 {
		// It's a sub-tag inside a tool tag
		f.inSubTag = true
		f.currentSubTag = tag

		// Skip requires_approval tag
		if isHiddenTag(tag) {
			// Don't show this tag or its content
			f.tagStack = append(f.tagStack, tag) // Still need to push to stack
			return
		}

		// Add prefix based on tool name and tag type
		f.buffer.WriteString(toolTagPrefix(f.tagStack[0], tag))

		// Apply color based on sub-tag type
		if tag == "path" {
			f.buffer.WriteString(core.ColorGreen)
		} else if tag == "command" {
			f.buffer.WriteString(core.ColorYellow)
		} else if tag == "content" {
			//f.buffer.WriteString(core.ColorBlue)
		}
	} else if !f.inToolTag {
		// For tags outside tool tags, output the tag
		f.buffer.WriteByte('<')
		f.buffer.WriteString(tag)
		f.buffer.WriteByte('>')
	}

	// Push the tag to stack
	f.tagStack = append(f.tagStack, tag)
}

func toolTagPrefix(tool string, tag string) string {
	switch tool {
	case "execute_command":
		if tag == "command" {
			return "Execute command: "
		}
	case "read_file":
		if tag == "path" {
			return "Read file: "
		}
	case "write_to_file":
		if tag == "path" {
			return "Write to file: "
		}
	case "replace_in_file":
		if tag == "path" {
			return "Replace in file: "
		}
	case "search_files":
		if tag == "path" {
			return "Search files: "
		}
	case "list_files":
		if tag == "path" {
			return "List files: "
		}
	case "list_code_definition_names":
		if tag == "path" {
			return "Code file: "
		}
	}

	return ""
}

// Check if a tag is a tool tag
func isToolTag(tag string) bool {
	toolTags := []string{
		"thinking",
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
	}

	for _, toolTag := range toolTags {
		if tag == toolTag {
			return true
		}
	}

	return false
}

// Check if a tag should be hidden
func isHiddenTag(tag string) bool {
	hiddenTags := []string{"requires_approval", "recursive", "regex", "file_pattern"}
	for _, hiddenTag := range hiddenTags {
		if tag == hiddenTag {
			return true
		}
	}
	return false
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
