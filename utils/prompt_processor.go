package utils

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"golang.org/x/net/html"
)

// ProcessPrompt processes user's prompt, finds text wrapped in backticks and appends the content
// If the text is a file path, it reads the file content and appends it
// If the text is a URL, it fetches the web content and appends it
func ProcessPrompt(prompt string) (string, error) {
	// Regular expression to match content wrapped in backticks
	re := regexp.MustCompile("`([^`]+)`")
	matches := re.FindAllStringSubmatch(prompt, -1)

	// If no matches found, return the original prompt
	if len(matches) == 0 {
		return prompt, nil
	}

	// Process each match
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		content := match[1]
		appendContent := ""
		var err error

		// Determine if the content is a file path or URL
		if IsURL(content) {
			// Process URL
			appendContent, err = FetchWebContent(content)
			if err != nil {
				return "", fmt.Errorf("failed to fetch web content: %v", err)
			}
			appendContent = "Web content:\n" + appendContent
		} else {
			// Process file path
			appendContent, err = readFileContent(content)
			if err != nil {
				return "", fmt.Errorf("failed to read file content: %v", err)
			}
			appendContent = "File content:\n" + appendContent
		}

		// Append the content to the prompt instead of replacing
		prompt = strings.Replace(prompt, match[0], match[0]+"\n\n"+appendContent+"\n\n", 1)
	}

	return prompt, nil
}

func HasBackticks(prompt string) bool {
	// Regular expression to match content wrapped in backticks
	re := regexp.MustCompile("`([^`]+)`")
	matches := re.FindAllStringSubmatch(prompt, -1)

	return len(matches) > 0
}

// IsURL determines if a string is a URL
func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// readFileContent reads the content of a file
func readFileContent(filePath string) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", absPath)
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// FetchWebContent gets web content and filters HTML tags
func FetchWebContent(urlStr string) (string, error) {
	// Create a cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create cookie jar: %v", err)
	}

	// Create a new HTTP client with timeout and redirect handling
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}

			// Copy headers to redirected request
			for key, val := range via[0].Header {
				if key != "Authorization" && key != "Cookie" {
					req.Header[key] = val
				}
			}
			return nil
		},
		Jar: jar,
	}

	// Create a new request
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	// Set headers to mimic Chrome browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Sec-Ch-Ua", "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"120\", \"Google Chrome\";v=\"120\"")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "\"macOS\"")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")

	// Send HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed, status code: %d", resp.StatusCode)
	}

	// Handle compressed content
	var reader io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer reader.(*gzip.Reader).Close()
	case "br":
		reader = brotli.NewReader(resp.Body)
	default:
		reader = resp.Body
	}

	// Parse HTML
	doc, err := html.Parse(reader)
	if err != nil {
		return "", err
	}

	// Extract text content
	var textContent strings.Builder
	extractText(doc, &textContent)

	// Clean up the text by removing excessive whitespace
	result := cleanText(textContent.String())

	return result, nil
}

// extractText recursively extracts text content from HTML nodes
func extractText(n *html.Node, sb *strings.Builder) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			sb.WriteString(text)
			sb.WriteString(" ") // Use space instead of newline to keep text flowing
		}
	}

	// Skip script and style tags
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "noscript" || n.Data == "iframe") {
		return
	}

	// Add structure based on HTML elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			sb.WriteString("\n\n")
		case "p", "div", "section", "article", "header", "footer", "main", "aside":
			sb.WriteString("\n")
		case "br":
			sb.WriteString("\n")
		case "li":
			sb.WriteString("\n- ")
		case "a":
			// Extract href attribute for links
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					// Process child nodes first to get the link text
					linkTextBuilder := &strings.Builder{}
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						extractText(c, linkTextBuilder)
					}
					linkText := strings.TrimSpace(linkTextBuilder.String())

					if linkText != "" {
						sb.WriteString(linkText)
						sb.WriteString(" [")
						sb.WriteString(attr.Val)
						sb.WriteString("] ")
					}
					return
				}
			}
		}
	}

	// Process child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, sb)
	}

	// Add appropriate spacing after block elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			sb.WriteString("\n\n")
		case "p", "div", "section", "article", "header", "footer", "main", "aside", "ul", "ol", "table":
			sb.WriteString("\n")
		}
	}
}

// cleanText removes excessive whitespace and normalizes line breaks
func cleanText(text string) string {
	// Replace multiple spaces with a single space
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Replace multiple newlines with a maximum of two
	re = regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}
