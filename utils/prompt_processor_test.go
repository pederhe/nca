package utils

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"golang.org/x/net/html"
)

func TestProcessPrompt(t *testing.T) {
	// Create temporary test file
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testFileContent := "This is test file content"
	err := os.WriteFile(testFilePath, []byte(testFileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup test HTTP server
	server := setupTestServer()
	defer server.Close()

	// Test cases
	tests := []struct {
		name             string
		prompt           string
		shouldContain    []string
		shouldNotContain []string
		expectError      bool
	}{
		{
			name:             "No special markers",
			prompt:           "This is a normal prompt",
			shouldContain:    []string{"This is a normal prompt"},
			shouldNotContain: []string{},
			expectError:      false,
		},
		{
			name:             "Contains file path",
			prompt:           "Please read this file: `" + testFilePath + "`",
			shouldContain:    []string{testFilePath, testFileContent},
			shouldNotContain: []string{},
			expectError:      false,
		},
		{
			name:             "Contains non-existent file path",
			prompt:           "Please read this file: `/non-existent-file.txt`",
			shouldContain:    []string{},
			shouldNotContain: []string{},
			expectError:      true,
		},
		{
			name:             "Contains URL to plain HTML",
			prompt:           "Please check this URL: `" + server.URL + "/plain`",
			shouldContain:    []string{"Test HTML Content", "This is a test paragraph"},
			shouldNotContain: []string{},
			expectError:      false,
		},
		{
			name:             "Contains URL to gzipped HTML",
			prompt:           "Please check this URL: `" + server.URL + "/gzip`",
			shouldContain:    []string{"Gzipped HTML Content", "This is a compressed paragraph"},
			shouldNotContain: []string{},
			expectError:      false,
		},
		{
			name:             "Contains URL to brotli HTML",
			prompt:           "Please check this URL: `" + server.URL + "/brotli`",
			shouldContain:    []string{"Brotli HTML Content", "This is a brotli compressed paragraph"},
			shouldNotContain: []string{},
			expectError:      false,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessPrompt(tt.prompt)

			// Check for errors
			if (err != nil) != tt.expectError {
				t.Errorf("ProcessPrompt() error = %v, expected error = %v", err, tt.expectError)
				return
			}

			// If expecting an error, don't check the result
			if tt.expectError {
				return
			}

			// Check that result contains all required strings
			for _, str := range tt.shouldContain {
				if !strings.Contains(result, str) {
					t.Errorf("ProcessPrompt() result does not contain expected string: %v", str)
				}
			}

			// Check that result does not contain any forbidden strings
			for _, str := range tt.shouldNotContain {
				if strings.Contains(result, str) {
					t.Errorf("ProcessPrompt() result contains forbidden string: %v", str)
				}
			}
		})
	}
}

// TestFetchWebContent tests the fetchWebContent function directly
func TestFetchWebContent(t *testing.T) {
	// Setup test HTTP server
	server := setupTestServer()
	defer server.Close()

	// Test cases
	tests := []struct {
		name          string
		url           string
		shouldContain []string
		expectError   bool
	}{
		{
			name:          "Plain HTML",
			url:           server.URL + "/plain",
			shouldContain: []string{"Test HTML Content", "This is a test paragraph"},
			expectError:   false,
		},
		{
			name:          "Gzipped HTML",
			url:           server.URL + "/gzip",
			shouldContain: []string{"Gzipped HTML Content", "This is a compressed paragraph"},
			expectError:   false,
		},
		{
			name:          "Brotli HTML",
			url:           server.URL + "/brotli",
			shouldContain: []string{"Brotli HTML Content", "This is a brotli compressed paragraph"},
			expectError:   false,
		},
		{
			name:          "404 Not Found",
			url:           server.URL + "/notfound",
			shouldContain: []string{},
			expectError:   true,
		},
		{
			name:          "Invalid URL",
			url:           "http://invalid.url.that.does.not.exist.example",
			shouldContain: []string{},
			expectError:   true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fetchWebContent(tt.url)

			// Check for errors
			if (err != nil) != tt.expectError {
				t.Errorf("fetchWebContent() error = %v, expected error = %v", err, tt.expectError)
				return
			}

			// If expecting an error, don't check the result
			if tt.expectError {
				return
			}

			// Check that result contains all required strings
			for _, str := range tt.shouldContain {
				if !strings.Contains(result, str) {
					t.Errorf("fetchWebContent() result does not contain expected string: %v", str)
				}
			}
		})
	}
}

// TestExtractText tests the extractText function
func TestExtractText(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Test Heading</h1>
    <p>This is a paragraph with <a href="https://example.com">a link</a>.</p>
    <ul>
        <li>Item 1</li>
        <li>Item 2</li>
    </ul>
    <script>console.log("This should be ignored");</script>
    <style>.hidden { display: none; }</style>
</body>
</html>
`
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	var sb strings.Builder
	extractText(doc, &sb)
	result := sb.String()

	expectedStrings := []string{
		"Test Heading",
		"This is a paragraph with",
		"a link",
		"Item 1",
		"Item 2",
	}

	unexpectedStrings := []string{
		"console.log",
		"This should be ignored",
		".hidden { display: none; }",
	}

	for _, str := range expectedStrings {
		if !strings.Contains(result, str) {
			t.Errorf("extractText() result does not contain expected string: %v", str)
		}
	}

	for _, str := range unexpectedStrings {
		if strings.Contains(result, str) {
			t.Errorf("extractText() result contains unexpected string: %v", str)
		}
	}
}

// setupTestServer creates a test HTTP server that serves different types of content
func setupTestServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request has Chrome-like headers
		userAgent := r.Header.Get("User-Agent")
		if !strings.Contains(userAgent, "Chrome") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Expected Chrome User-Agent"))
			return
		}

		switch r.URL.Path {
		case "/plain":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Test HTML Content</title>
				</head>
				<body>
					<h1>Test HTML Content</h1>
					<p>This is a test paragraph</p>
					<a href="https://example.com">Example Link</a>
				</body>
				</html>
			`))

		case "/gzip":
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)

			// Create gzipped content
			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			gzipWriter.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Gzipped HTML Content</title>
				</head>
				<body>
					<h1>Gzipped HTML Content</h1>
					<p>This is a compressed paragraph</p>
					<a href="https://example.com">Example Link</a>
				</body>
				</html>
			`))
			gzipWriter.Close()
			w.Write(buf.Bytes())

		case "/brotli":
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "br")
			w.WriteHeader(http.StatusOK)

			// Create brotli compressed content
			var buf bytes.Buffer
			brotliWriter := brotli.NewWriter(&buf)
			brotliWriter.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Brotli HTML Content</title>
				</head>
				<body>
					<h1>Brotli HTML Content</h1>
					<p>This is a brotli compressed paragraph</p>
					<a href="https://example.com">Example Link</a>
				</body>
				</html>
			`))
			brotliWriter.Close()
			w.Write(buf.Bytes())

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}
	}))

	return server
}
