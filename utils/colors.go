package utils

import (
	"os"
)

// ANSI color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorRed    = "\033[31m"
	ColorCyan   = "\033[36m"
)

// IsOutputPiped detects whether standard output is redirected through a pipe
func IsOutputPiped() bool {
	stat, _ := os.Stdout.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// GetColor returns the appropriate color code based on the output target
// If output is to a pipe, returns an empty string, otherwise returns the color code
func GetColor(color string) string {
	if IsOutputPiped() {
		return ""
	}
	return color
}

// ColoredText returns colored text, if output is to a pipe, no color will be added
func ColoredText(text string, color string) string {
	if IsOutputPiped() {
		return text
	}
	return color + text + ColorReset
}
