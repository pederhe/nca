package core

import (
	"reflect"
	"testing"
)

func TestParseToolUse_SingleTool(t *testing.T) {
	// Test case for a single tool (execute_command)
	content := `I'll execute the command to list files.

<execute_command>
<command>ls -la</command>
<requires_approval>true</requires_approval>
</execute_command>
`
	result := ParseToolUse(content)

	// Check if the tool was correctly parsed
	if result["tool"] != "execute_command" {
		t.Errorf("Expected tool to be 'execute_command', got %v", result["tool"])
	}

	// Check command parameter
	if result["command"] != "ls -la" {
		t.Errorf("Expected command to be 'ls -la', got %v", result["command"])
	}

	// Check requires_approval parameter
	if result["requires_approval"] != true {
		t.Errorf("Expected requires_approval to be true, got %v", result["requires_approval"])
	}

	// Check that has_multiple_tools is not set
	if _, exists := result["has_multiple_tools"]; exists {
		t.Errorf("Expected has_multiple_tools to not be set, but it was")
	}
}

func TestParseToolUse_MultipleTool(t *testing.T) {
	// Test case for multiple tools
	content := `I'll first read a file and then execute a command.

<read_file>
<path>/etc/passwd</path>
</read_file>

Now I'll execute a command:

<execute_command>
<command>ls -la</command>
<requires_approval>true</requires_approval>
</execute_command>
`
	result := ParseToolUse(content)

	// Check if the first tool was correctly parsed
	if result["tool"] != "read_file" {
		t.Errorf("Expected tool to be 'read_file', got %v", result["tool"])
	}

	// Check path parameter
	if result["path"] != "/etc/passwd" {
		t.Errorf("Expected path to be '/etc/passwd', got %v", result["path"])
	}

	// Check that has_multiple_tools is set
	if hasMultiple, exists := result["has_multiple_tools"]; !exists || hasMultiple != true {
		t.Errorf("Expected has_multiple_tools to be true, got %v", hasMultiple)
	}

	// Check detected_tools
	expectedTools := "read_file, execute_command"
	if detectedTools, exists := result["detected_tools"].(string); !exists || detectedTools != expectedTools {
		t.Errorf("Expected detected_tools to be '%s', got '%v'", expectedTools, detectedTools)
	}
}

func TestParseToolUse_WriteToFile(t *testing.T) {
	// Test case for write_to_file tool
	content := `I'll write to a file.

<write_to_file>
<path>test.txt</path>
<content>This is a test content
with multiple lines
that should be preserved.</content>
</write_to_file>
`
	result := ParseToolUse(content)

	// Check if the tool was correctly parsed
	if result["tool"] != "write_to_file" {
		t.Errorf("Expected tool to be 'write_to_file', got %v", result["tool"])
	}

	// Check path parameter
	if result["path"] != "test.txt" {
		t.Errorf("Expected path to be 'test.txt', got %v", result["path"])
	}

	// Check content parameter (should preserve formatting)
	expectedContent := "This is a test content\nwith multiple lines\nthat should be preserved."
	if result["content"] != expectedContent {
		t.Errorf("Expected content to be '%s', got '%v'", expectedContent, result["content"])
	}
}

func TestParseToolUse_GitCommit(t *testing.T) {
	// Test case for git_commit tool
	content := `I'll commit the changes.

<git_commit>
<message>Fix bug in login functionality</message>
<files>
src/login.js
src/auth.js
test/login_test.js
</files>
</git_commit>
`
	result := ParseToolUse(content)

	// Check if the tool was correctly parsed
	if result["tool"] != "git_commit" {
		t.Errorf("Expected tool to be 'git_commit', got %v", result["tool"])
	}

	// Check message parameter
	if result["message"] != "Fix bug in login functionality" {
		t.Errorf("Expected message to be 'Fix bug in login functionality', got %v", result["message"])
	}

	// Check files parameter
	expectedFiles := []string{"src/login.js", "src/auth.js", "test/login_test.js"}
	files, ok := result["files"].([]string)
	if !ok {
		t.Errorf("Expected files to be a string slice, got %T", result["files"])
	} else if !reflect.DeepEqual(files, expectedFiles) {
		t.Errorf("Expected files to be %v, got %v", expectedFiles, files)
	}
}

func TestParseToolUse_SearchFiles(t *testing.T) {
	// Test case for search_files tool
	content := `I'll search for files.

<search_files>
<path>src</path>
<regex>func.*Init</regex>
<file_pattern>*.go</file_pattern>
</search_files>
`
	result := ParseToolUse(content)

	// Check if the tool was correctly parsed
	if result["tool"] != "search_files" {
		t.Errorf("Expected tool to be 'search_files', got %v", result["tool"])
	}

	// Check path parameter
	if result["path"] != "src" {
		t.Errorf("Expected path to be 'src', got %v", result["path"])
	}

	// Check regex parameter
	if result["regex"] != "func.*Init" {
		t.Errorf("Expected regex to be 'func.*Init', got %v", result["regex"])
	}

	// Check file_pattern parameter
	if result["file_pattern"] != "*.go" {
		t.Errorf("Expected file_pattern to be '*.go', got %v", result["file_pattern"])
	}
}

func TestParseToolUse_NoTool(t *testing.T) {
	// Test case for no tool
	content := `I don't know how to help with that.`
	result := ParseToolUse(content)

	// Check if the result is nil
	if result != nil {
		t.Errorf("Expected result to be nil, got %v", result)
	}
}

func TestParseToolUse_InvalidTool(t *testing.T) {
	// Test case for invalid tool (missing closing tag)
	content := `<execute_command>
<command>ls -la</command>
`
	result := ParseToolUse(content)

	// Check if the result is nil
	if result != nil {
		t.Errorf("Expected result to be nil, got %v", result)
	}
}

func TestRemoveThinkingTags(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No thinking tags",
			input:    "This is a normal response without thinking tags.",
			expected: "This is a normal response without thinking tags.",
		},
		{
			name:     "Single thinking tag",
			input:    "Let me think about this. <thinking>I need to consider the options carefully.</thinking> I think we should use option A.",
			expected: "Let me think about this.  I think we should use option A.",
		},
		{
			name:     "Multiple thinking tags",
			input:    "<thinking>First, I'll analyze the problem.</thinking> The issue is X. <thinking>Now I need to consider solutions.</thinking> We can solve it with Y.",
			expected: " The issue is X.  We can solve it with Y.",
		},
		{
			name:     "Nested thinking tags (not supported by regex)",
			input:    "<thinking>Outer thinking <thinking>Inner thinking</thinking> still outer</thinking> Normal text",
			expected: " Normal text",
		},
		{
			name:     "Multiline thinking tags",
			input:    "Start\n<thinking>\nThis is line 1\nThis is line 2\n</thinking>\nEnd",
			expected: "Start\n\nEnd",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := RemoveThinkingTags(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestParseToolUse_AttemptCompletion(t *testing.T) {
	// Test case for attempt_completion tool
	content := `I'll complete the task.

<attempt_completion>
<command>echo "Task completed"</command>
<r>The task has been completed successfully.</r>
</attempt_completion>
`
	result := ParseToolUse(content)

	// Check if the tool was correctly parsed
	if result["tool"] != "attempt_completion" {
		t.Errorf("Expected tool to be 'attempt_completion', got %v", result["tool"])
	}

	// Check command parameter
	if result["command"] != "echo \"Task completed\"" {
		t.Errorf("Expected command to be 'echo \"Task completed\"', got %v", result["command"])
	}

	// Check result parameter
	if result["result"] != "The task has been completed successfully." {
		t.Errorf("Expected result to be 'The task has been completed successfully.', got %v", result["result"])
	}
}

func TestParseToolUse_ReplaceInFile(t *testing.T) {
	// Test case for replace_in_file tool
	content := `I'll replace content in the file.

<replace_in_file>
<path>src/main.go</path>
<diff><<<<<<< SEARCH
func main() {
	fmt.Println("Hello, World!")
}
=======
func main() {
	fmt.Println("Hello, Updated World!")
}
>>>>>>> REPLACE</diff>
</replace_in_file>
`
	result := ParseToolUse(content)

	// Check if the tool was correctly parsed
	if result["tool"] != "replace_in_file" {
		t.Errorf("Expected tool to be 'replace_in_file', got %v", result["tool"])
	}

	// Check path parameter
	if result["path"] != "src/main.go" {
		t.Errorf("Expected path to be 'src/main.go', got %v", result["path"])
	}

	// Check diff parameter (should preserve formatting)
	expectedDiff := `<<<<<<< SEARCH
func main() {
	fmt.Println("Hello, World!")
}
=======
func main() {
	fmt.Println("Hello, Updated World!")
}
>>>>>>> REPLACE`
	if result["diff"] != expectedDiff {
		t.Errorf("Expected diff to be '%s', got '%v'", expectedDiff, result["diff"])
	}
}
