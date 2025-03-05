package core

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ExecuteCommand executes a command line command
func ExecuteCommand(params map[string]interface{}) string {
	command, ok := params["command"].(string)
	if !ok {
		return "Error: Missing command parameter"
	}

	requiresApproval, _ := params["requires_approval"].(bool)
	if requiresApproval {
		fmt.Printf("Need to execute command: %s\nContinue? (y/n): ", command)
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

	cmd := exec.Command(parts[0], parts[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
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

	fileContent := string(content)

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

	return fmt.Sprintf("File successfully updated: %s", path)
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