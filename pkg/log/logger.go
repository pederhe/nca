package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Debug mode variables
var (
	debugMode    bool
	debugLogFile *os.File
	sessionID    string
	debugLogPath string
)

// InitDebugMode initializes debug mode, creating necessary directories and log file
func InitDebugMode() {
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

	// Set debug mode to true
	debugMode = true

	// Log session start
	LogDebug(fmt.Sprintf("Session started at %s\n", now.Format(time.RFC3339)))
}

// LogDebug writes a message to the debug log
func LogDebug(message string) {
	if !debugMode || debugLogFile == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, message)

	if _, err := debugLogFile.WriteString(logEntry); err != nil {
		fmt.Printf("Warning: Failed to write to debug log: %s\n", err)
	}
}

// CloseDebugLog closes the debug log file
func CloseDebugLog() {
	if debugLogFile != nil {
		LogDebug("Session ended\n")
		debugLogFile.Close()
		debugLogFile = nil
	}
}

// IsDebugMode returns whether debug mode is enabled
func IsDebugMode() bool {
	return debugMode
}

// EnableDebugMode enables debug mode programmatically
func EnableDebugMode() {
	if !debugMode {
		debugMode = true
		InitDebugMode()
	}
}

// GetDebugLogPath returns the path to the current debug log file
func GetDebugLogPath() string {
	return debugLogPath
}
