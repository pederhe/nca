package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pederhe/nca/config"
	"github.com/pederhe/nca/utils"
)

// ExecuteCommand executes a command line command
func ExecuteCommand(params map[string]interface{}) string {
	command, ok := params["command"].(string)
	if !ok {
		return "Error: Missing command parameter"
	}

	autoApprove := config.Get("auto_approve") == "true" || config.Get("auto_approve") == "1"
	requiresApproval, _ := params["requires_approval"].(bool)
	if !autoApprove && requiresApproval {
		fmt.Printf("Need to execute command: %s\nContinue? (y/n): ", utils.ColoredText(command, utils.ColorYellow))
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			return "Command execution cancelled"
		}
	}

	// Split command and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "Error: Empty command"
	}
	if strings.Contains(command, ";") {
		parts = []string{"bash", "-c", command}
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = os.Environ()

	err := cmd.Run()
	if err != nil || stderr.Len() > 0 {
		return fmt.Sprintf("Command execution error: %s\n%s", err, stderr.String())
	}

	return stdout.String()
}

// ReadFile reads the contents of a file
func ReadFile(params map[string]interface{}) string {
	path, ok := params["path"].(string)
	if !ok {
		return "Error: Missing file path parameter"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading file: %s", err)
	}

	return string(data)
}

// WriteToFile writes content to a file
func WriteToFile(params map[string]interface{}) string {
	path, ok := params["path"].(string)
	if !ok {
		return "Error: Missing file path parameter"
	}

	content, ok := params["content"].(string)
	if !ok {
		return "Error: Missing file content parameter"
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %s", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %s", err)
	}

	return fmt.Sprintf("File successfully written: %s", path)
}

// ReplaceInFile replaces content in a file
func ReplaceInFile(params map[string]interface{}) string {
	path, ok := params["path"].(string)
	if !ok {
		return "Error: Missing file path parameter"
	}

	diff, ok := params["diff"].(string)
	if !ok {
		return "Error: Missing diff parameter"
	}

	// Read original file content
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading file: %s", err)
	}

	originalContent := string(content)
	fileContent := originalContent

	// Parse and apply SEARCH/REPLACE blocks
	// Use regex to match SEARCH/REPLACE blocks
	re := regexp.MustCompile(`<<<<<<< SEARCH\n([\s\S]*?)\n=======\n([\s\S]*?)\n>>>>>>> REPLACE`)
	matches := re.FindAllStringSubmatch(diff, -1)

	if len(matches) == 0 {
		return "Error: No valid SEARCH/REPLACE blocks found"
	}

	// Apply each SEARCH/REPLACE block
	for _, match := range matches {
		search := match[1]
		replace := match[2]
		fileContent = strings.Replace(fileContent, search, replace, 1)
	}

	// Write back to file
	if err := os.WriteFile(path, []byte(fileContent), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %s", err)
	}

	// Generate diff output in git style
	diffOutput := generateGitStyleDiff(path, originalContent, fileContent)

	return fmt.Sprintf("File successfully updated: %s\n%s", path, diffOutput)
}

// generateGitStyleDiff generates a git-style diff between original and new content
func generateGitStyleDiff(filename string, originalContent, newContent string) string {
	// Create temporary files to store original and new content
	tempDir, err := os.MkdirTemp("", "nca-diff")
	if err != nil {
		return fmt.Sprintf("Error creating temp directory: %s", err)
	}
	defer os.RemoveAll(tempDir)

	originalFile := filepath.Join(tempDir, "original")
	newFile := filepath.Join(tempDir, "new")

	if err := os.WriteFile(originalFile, []byte(originalContent), 0644); err != nil {
		return fmt.Sprintf("Error writing temp file: %s", err)
	}

	if err := os.WriteFile(newFile, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error writing temp file: %s", err)
	}

	// Use external diff command to generate standard diff output
	cmd := exec.Command("diff", "-u", originalFile, newFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// diff command returns non-zero exit code when differences are found, which is normal
	_ = cmd.Run()

	if stderr.Len() > 0 {
		return fmt.Sprintf("Error generating diff: %s", stderr.String())
	}

	diffOutput := stdout.String()
	if diffOutput == "" {
		return "No changes detected"
	}

	// Process diff output, replace temporary file paths with actual file path
	diffOutput = strings.ReplaceAll(diffOutput, "--- "+originalFile, "--- a/"+filename)
	diffOutput = strings.ReplaceAll(diffOutput, "+++ "+newFile, "+++ b/"+filename)

	// Add colors
	var coloredOutput strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(diffOutput))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			coloredOutput.WriteString(fmt.Sprintf("%s\n", utils.ColoredText(line, utils.ColorGreen)))
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			coloredOutput.WriteString(fmt.Sprintf("%s\n", utils.ColoredText(line, utils.ColorRed)))
		} else if strings.HasPrefix(line, "@@") {
			coloredOutput.WriteString(fmt.Sprintf("%s\n", utils.ColoredText(line, utils.ColorCyan)))
		} else {
			coloredOutput.WriteString(line + "\n")
		}
	}

	return coloredOutput.String()
}

// SearchFiles searches for content in files
func SearchFiles(params map[string]interface{}) string {
	path, ok := params["path"].(string)
	if !ok {
		return "Error: Missing directory path parameter"
	}

	regexStr, ok := params["regex"].(string)
	if !ok {
		return "Error: Missing regex parameter"
	}

	filePattern, _ := params["file_pattern"].(string)
	if filePattern == "" {
		filePattern = "*"
	}

	// Compile regex
	re, err := regexp.Compile(regexStr)
	if err != nil {
		return fmt.Sprintf("Error compiling regex: %s", err)
	}

	var results strings.Builder
	results.WriteString(fmt.Sprintf("Searching for '%s' in '%s' (pattern: %s)\n\n", regexStr, path, filePattern))

	// Walk through directory
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches pattern
		matched, err := filepath.Match(filePattern, filepath.Base(filePath))
		if err != nil || !matched {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}

		fileContent := string(content)
		lines := strings.Split(fileContent, "\n")

		// Find matches
		matches := re.FindAllStringIndex(fileContent, -1)
		if len(matches) > 0 {
			relPath, _ := filepath.Rel(path, filePath)
			results.WriteString(fmt.Sprintf("File: %s\n", relPath))

			for _, match := range matches {
				start, end := match[0], match[1]

				// Find line numbers
				lineStart := strings.Count(fileContent[:start], "\n")
				lineEnd := lineStart + strings.Count(fileContent[start:end], "\n")

				// Get context (3 lines before and after)
				contextStart := max(0, lineStart-3)
				contextEnd := min(len(lines), lineEnd+4)

				results.WriteString(fmt.Sprintf("Match at lines %d-%d:\n", lineStart+1, lineEnd+1))

				// Print context
				for i := contextStart; i < contextEnd; i++ {
					prefix := "  "
					if i >= lineStart && i <= lineEnd {
						prefix = "> "
					}
					results.WriteString(fmt.Sprintf("%s%4d: %s\n", prefix, i+1, lines[i]))
				}

				results.WriteString("\n")
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Sprintf("Error searching files: %s", err)
	}

	if results.Len() == 0 {
		return "No matches found"
	}

	return results.String()
}

// ListFiles lists files in a directory
func ListFiles(params map[string]interface{}) string {
	path, ok := params["path"].(string)
	if !ok {
		return "Error: Missing directory path parameter"
	}

	recursive, _ := params["recursive"].(bool)
	var files strings.Builder
	var recursiveText string
	if recursive {
		recursiveText = " (recursive)"
	} else {
		recursiveText = ""
	}
	files.WriteString(fmt.Sprintf("Listing files in '%s'%s:\n\n", path, recursiveText))

	if recursive {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, _ := filepath.Rel(path, filePath)
			if relPath == "." {
				return nil
			}

			if info.IsDir() {
				files.WriteString(fmt.Sprintf("📁 %s/\n", relPath))
			} else {
				files.WriteString(fmt.Sprintf("📄 %s (%d bytes)\n", relPath, info.Size()))
			}

			return nil
		})

		if err != nil {
			return fmt.Sprintf("Error listing files: %s", err)
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Sprintf("Error listing files: %s", err)
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if entry.IsDir() {
				files.WriteString(fmt.Sprintf("📁 %s/\n", entry.Name()))
			} else {
				files.WriteString(fmt.Sprintf("📄 %s (%d bytes)\n", entry.Name(), info.Size()))
			}
		}
	}

	if files.Len() == 0 {
		return "No files found"
	}

	return files.String()
}

// ListCodeDefinitionNames lists code definitions in a directory
func ListCodeDefinitionNames(params map[string]interface{}) string {
	path, ok := params["path"].(string)
	if !ok {
		return "Error: Missing directory path parameter"
	}

	var definitions strings.Builder
	definitions.WriteString(fmt.Sprintf("Listing code definitions in '%s':\n\n", path))

	err := filepath.Walk(path, func(filePath string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Check if it's a code file
		ext := filepath.Ext(filePath)
		if !isCodeFile(ext) {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}

		// Extract definitions
		defs := extractDefinitions(string(content), ext)
		if len(defs) > 0 {
			relPath, _ := filepath.Rel(path, filePath)
			definitions.WriteString(fmt.Sprintf("File: %s\n", relPath))
			definitions.WriteString("Definitions:\n")
			for _, def := range defs {
				definitions.WriteString(fmt.Sprintf("  - %s\n", def))
			}
			definitions.WriteString("\n")
		}

		return nil
	})

	if err != nil {
		return fmt.Sprintf("Error listing definitions: %s", err)
	}

	if definitions.Len() == 0 {
		return "No code definitions found"
	}

	return definitions.String()
}

// Helper functions

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isCodeFile(ext string) bool {
	codeExts := map[string]bool{
		".go":   true,
		".js":   true,
		".ts":   true,
		".py":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".cs":   true,
		".php":  true,
		".rb":   true,
		".rs":   true,
	}
	return codeExts[ext]
}

func extractDefinitions(content, ext string) []string {
	var definitions []string

	switch ext {
	case ".go":
		// Match Go functions and type definitions
		funcRe := regexp.MustCompile(`func\s+([A-Za-z0-9_]+)`)
		typeRe := regexp.MustCompile(`type\s+([A-Za-z0-9_]+)`)

		funcMatches := funcRe.FindAllStringSubmatch(content, -1)
		for _, match := range funcMatches {
			definitions = append(definitions, "func "+match[1])
		}

		typeMatches := typeRe.FindAllStringSubmatch(content, -1)
		for _, match := range typeMatches {
			definitions = append(definitions, "type "+match[1])
		}

	case ".js", ".ts":
		// Match JavaScript/TypeScript functions and class definitions
		funcRe := regexp.MustCompile(`function\s+([A-Za-z0-9_]+)`)
		classRe := regexp.MustCompile(`class\s+([A-Za-z0-9_]+)`)

		funcMatches := funcRe.FindAllStringSubmatch(content, -1)
		for _, match := range funcMatches {
			definitions = append(definitions, "function "+match[1])
		}

		classMatches := classRe.FindAllStringSubmatch(content, -1)
		for _, match := range classMatches {
			definitions = append(definitions, "class "+match[1])
		}

		// Can add support for more languages
	}

	return definitions
}

func FollowupQuestion(params map[string]interface{}) string {
	// Get the question from the tool use parameters
	question, ok := params["question"].(string)
	if !ok || question == "" {
		return "Error: No question provided for ask_followup_question tool"
	}

	return ""
}

// PlanModeResponse handles responses in plan mode
func PlanModeResponse(params map[string]interface{}) string {
	// Get the response content from the tool use parameters
	response, ok := params["response"].(string)
	if !ok || response == "" {
		return "Error: No response provided for plan_mode_response tool"
	}

	return response
}

// GitCommit handles the git_commit tool functionality
func GitCommit(params map[string]interface{}) string {
	// Extract parameters
	commitMessage, ok := params["message"].(string)
	if !ok || commitMessage == "" {
		return "Error: message parameter is required for git_commit"
	}

	// Extract files parameter
	var modifiedFiles []string
	if filesParam, ok := params["files"].([]string); ok && len(filesParam) > 0 {
		modifiedFiles = filesParam
	}

	// Validate parameters
	if len(modifiedFiles) == 0 {
		return "Error: files parameter is required for git_commit"
	}

	// Display files to be committed
	fmt.Println("Files to be committed:")
	for _, file := range modifiedFiles {
		fmt.Printf("  %s%s%s\n", utils.ColorGreen, file, utils.ColorReset)
	}

	// Ask for confirmation to proceed with these files
	fmt.Print("Do you want to proceed with these files? (y/n): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		return "Commit cancelled"
	}

	fmt.Printf("Commit message: %s%s%s\n", utils.ColorYellow, commitMessage, utils.ColorReset)
	fmt.Print("Do you want to use this message? (y/n/custom): ")

	reader := bufio.NewReader(os.Stdin)
	response, _ = reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if strings.ToLower(response) == "n" {
		return "Commit cancelled"
	} else if strings.ToLower(response) != "y" {
		// User wants to provide a custom message
		fmt.Print("Enter your custom commit message: ")
		customMessage, _ := reader.ReadString('\n')
		customMessage = strings.TrimSpace(customMessage)

		if customMessage != "" {
			commitMessage = customMessage
		}
	}

	// Now execute the add and commit operations
	err := utils.GitAdd(modifiedFiles) // Add specified files
	if err != nil {
		return fmt.Sprintf("Error adding files to staging area: %s", err)
	}

	// Commit changes
	err = utils.GitCommit(commitMessage)
	if err != nil {
		return fmt.Sprintf("Error committing changes: %s", err)
	}

	return fmt.Sprintf("Successfully committed changes with message: %s", commitMessage)
}

// FetchWebContent fetches the content of a web page
func FetchWebContent(params map[string]interface{}) string {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return "Error: Missing or empty URL parameter"
	}

	// Validate URL format
	if !utils.IsURL(url) {
		return fmt.Sprintf("Error: Invalid URL format: %s", url)
	}

	fmt.Printf("Fetching web content from: %s\n", utils.ColoredText(url, utils.ColorYellow))

	// Fetch web content
	content, err := utils.FetchWebContent(url)
	if err != nil {
		return fmt.Sprintf("Error fetching web content: %s", err)
	}

	// Format the result
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Web content from %s:\n\n", url))
	result.WriteString(content)

	return result.String()
}
