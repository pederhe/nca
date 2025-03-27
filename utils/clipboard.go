package utils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// GetClipboardContent retrieves the content from the clipboard
func GetClipboardContent() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		// Prefer xclip as it's more commonly available on most Linux distributions
		cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// IsClipboardPrefix checks if the input is a prefix of the clipboard content
func IsClipboardPrefix(input string) (bool, string, error) {
	trimmedInput := strings.TrimSpace(input)
	if trimmedInput == "" {
		return false, "", nil
	}
	clipContent, err := GetClipboardContent()
	if err != nil {
		return false, "", err
	}

	trimmedClip := strings.TrimSpace(clipContent)

	if strings.HasPrefix(trimmedClip, trimmedInput) {
		return true, trimmedClip, nil
	}

	return false, trimmedClip, nil
}
