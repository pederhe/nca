package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Setup test environment with temporary directory and files
func setupTestEnv(t *testing.T) (string, func()) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "tools_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create a test file
	testFilePath := filepath.Join(tempDir, "test_file.txt")
	err = os.WriteFile(testFilePath, []byte("This is a test file content"), 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test code file
	testCodeFilePath := filepath.Join(tempDir, "test_code.go")
	err = os.WriteFile(testCodeFilePath, []byte(`package main

import "fmt"

func testFunction() {
	fmt.Println("Hello, World!")
}

type TestStruct struct {
	Name string
	Age  int
}
`), 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create test code file: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// Test ExecuteCommand function
func TestExecuteCommand(t *testing.T) {
	// Test simple command
	params := map[string]interface{}{
		"command":           "echo 'test command'",
		"requires_approval": false,
	}

	result := ExecuteCommand(params)
	assert.Contains(t, result, "test command")

	// Test invalid command
	params = map[string]interface{}{
		"command":           "invalid_command_that_doesnt_exist",
		"requires_approval": false,
	}

	result = ExecuteCommand(params)
	assert.Contains(t, result, "Command execution error")
}

// Test ReadFile function
func TestReadFile(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testFilePath := filepath.Join(tempDir, "test_file.txt")

	// Test valid file reading
	params := map[string]interface{}{
		"path": testFilePath,
	}

	result := ReadFile(params)
	assert.Equal(t, "This is a test file content", result)

	// Test invalid file path
	params = map[string]interface{}{
		"path": filepath.Join(tempDir, "non_existent_file.txt"),
	}

	result = ReadFile(params)
	assert.Contains(t, result, "Error reading file")
}

// Test WriteToFile function
func TestWriteToFile(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	newFilePath := filepath.Join(tempDir, "new_file.txt")
	fileContent := "This is new content"

	// Test writing to file
	params := map[string]interface{}{
		"path":    newFilePath,
		"content": fileContent,
	}

	result := WriteToFile(params)
	assert.Contains(t, result, "File successfully written")

	// Verify file content
	content, err := os.ReadFile(newFilePath)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, string(content))

	// Test creating nested directory structure
	nestedFilePath := filepath.Join(tempDir, "nested", "dir", "test.txt")
	params = map[string]interface{}{
		"path":    nestedFilePath,
		"content": fileContent,
	}

	result = WriteToFile(params)
	assert.Contains(t, result, "File successfully written")

	// Verify file content
	content, err = os.ReadFile(nestedFilePath)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, string(content))
}

// Test ReplaceInFile function
func TestReplaceInFile(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	testFilePath := filepath.Join(tempDir, "test_file.txt")

	// Create file with specific content
	initialContent := "This is line 1\nThis is line 2\nThis is line 3"
	err := os.WriteFile(testFilePath, []byte(initialContent), 0644)
	assert.NoError(t, err)

	// Test replacing file content
	diff := `<<<<<<< SEARCH
This is line 2
=======
This is replaced line 2
>>>>>>> REPLACE`

	params := map[string]interface{}{
		"path": testFilePath,
		"diff": diff,
	}

	result := ReplaceInFile(params)
	assert.Contains(t, result, "File successfully updated")

	// Verify file content
	content, err := os.ReadFile(testFilePath)
	assert.NoError(t, err)
	expectedContent := "This is line 1\nThis is replaced line 2\nThis is line 3"
	assert.Equal(t, expectedContent, string(content))

	// Test invalid replacement format
	params = map[string]interface{}{
		"path": testFilePath,
		"diff": "Invalid diff format",
	}

	result = ReplaceInFile(params)
	assert.Contains(t, result, "No valid SEARCH/REPLACE blocks found")
}

// Test SearchFiles function
func TestSearchFiles(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Test searching file content
	params := map[string]interface{}{
		"path":  tempDir,
		"regex": "test",
	}

	result := SearchFiles(params)
	assert.Contains(t, result, "test_file.txt")

	// Test searching with file pattern
	params = map[string]interface{}{
		"path":         tempDir,
		"regex":        "func",
		"file_pattern": "*.go",
	}

	result = SearchFiles(params)
	assert.Contains(t, result, "test_code.go")
	assert.Contains(t, result, "testFunction")
}

// Test ListFiles function
func TestListFiles(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Test listing files
	params := map[string]interface{}{
		"path": tempDir,
	}

	result := ListFiles(params)
	assert.Contains(t, result, "test_file.txt")
	assert.Contains(t, result, "test_code.go")

	// Test recursive file listing
	// Create subdirectory and file
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	assert.NoError(t, err)

	subFilePath := filepath.Join(subDir, "subfile.txt")
	err = os.WriteFile(subFilePath, []byte("Subdir file content"), 0644)
	assert.NoError(t, err)

	params = map[string]interface{}{
		"path":      tempDir,
		"recursive": true,
	}

	result = ListFiles(params)
	assert.Contains(t, result, "test_file.txt")
	assert.Contains(t, result, "subdir")
	assert.Contains(t, result, "subfile.txt")
}

// Test ListCodeDefinitionNames function
func TestListCodeDefinitionNames(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Test listing code definitions
	params := map[string]interface{}{
		"path": tempDir,
	}

	result := ListCodeDefinitionNames(params)
	assert.Contains(t, result, "test_code.go")
	assert.Contains(t, result, "func testFunction")
	assert.Contains(t, result, "type TestStruct")
}

// Test helper functions
func TestHelperFunctions(t *testing.T) {
	// Test max function
	assert.Equal(t, 5, max(3, 5))
	assert.Equal(t, 3, max(3, 1))

	// Test min function
	assert.Equal(t, 3, min(3, 5))
	assert.Equal(t, 1, min(3, 1))

	// Test isCodeFile function
	assert.True(t, isCodeFile(".go"))
	assert.True(t, isCodeFile(".js"))
	assert.False(t, isCodeFile(".txt"))

	// Test extractDefinitions function
	goCode := `
package test

func TestFunction() {
	fmt.Println("Test")
}

type TestType struct {
	Field string
}
`
	defs := extractDefinitions(goCode, ".go")
	assert.Contains(t, defs, "func TestFunction")
	assert.Contains(t, defs, "type TestType")

	jsCode := `
function jsFunction() {
	console.log("Test");
}

class JsClass {
	constructor() {
		this.field = "value";
	}
}
`
	defs = extractDefinitions(jsCode, ".js")
	assert.Contains(t, defs, "function jsFunction")
	assert.Contains(t, defs, "class JsClass")
}

// Test FetchWebContent function
func TestFetchWebContent(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Hello World</h1>
    <p>This is a test page for FetchWebContent</p>
</body>
</html>
			`))
		} else if r.URL.Path == "/binary" {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{0x00, 0x01, 0x02, 0x03})
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 - Not Found"))
		}
	}))
	defer server.Close()

	// Test successful HTML content retrieval
	params := map[string]interface{}{
		"url": server.URL + "/success",
	}

	result := FetchWebContent(params)
	assert.Contains(t, result, "Hello World")
	assert.Contains(t, result, "This is a test page for FetchWebContent")

	// Test 404 error page
	params = map[string]interface{}{
		"url": server.URL + "/not-found",
	}

	result = FetchWebContent(params)
	assert.Contains(t, result, "Error fetching web content")

	// Test invalid URL
	params = map[string]interface{}{
		"url": "invalid-url",
	}

	result = FetchWebContent(params)
	assert.Contains(t, result, "Invalid URL format")
}

// Test GitCommit function
func TestGitCommit(t *testing.T) {
	// Since GitCommit needs actual git repository and user interaction, we only test parameter validation here

	// Test missing message parameter
	params := map[string]interface{}{
		"files": []string{"file1.txt", "file2.txt"},
	}

	result := GitCommit(params)
	assert.Contains(t, result, "Error: message parameter is required")

	// Test missing files parameter
	params = map[string]interface{}{
		"message": "Test commit message",
	}

	result = GitCommit(params)
	assert.Contains(t, result, "Error: files parameter is required")
}

// Test FollowupQuestion function
func TestFollowupQuestion(t *testing.T) {
	params := map[string]interface{}{
		"question": "Test question?",
	}

	result := FollowupQuestion(params)
	assert.Equal(t, "", result)

	// Test missing question parameter
	params = map[string]interface{}{}
	result = FollowupQuestion(params)
	assert.Contains(t, result, "Error: No question provided")
}

// Test PlanModeResponse function
func TestPlanModeResponse(t *testing.T) {
	testResponse := "This is a test response"
	params := map[string]interface{}{
		"response": testResponse,
	}

	result := PlanModeResponse(params)
	assert.Equal(t, testResponse, result)

	// Test missing response parameter
	params = map[string]interface{}{}
	result = PlanModeResponse(params)
	assert.Contains(t, result, "Error: No response provided")
}
