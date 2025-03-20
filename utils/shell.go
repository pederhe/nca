package utils

import (
	"os"
	"runtime"

	"github.com/joho/godotenv"
)

// ShellPaths defines shell path constants for various platforms
var ShellPaths = struct {
	// Windows paths
	PowerShell7      string
	PowerShellLegacy string
	CMD              string
	WSLBash          string
	// Unix paths
	MacDefault   string
	LinuxDefault string
	CSH          string
	BASH         string
	KSH          string
	SH           string
	ZSH          string
	DASH         string
	TCSH         string
	Fallback     string
}{
	PowerShell7:      "C:\\Program Files\\PowerShell\\7\\pwsh.exe",
	PowerShellLegacy: "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
	CMD:              "C:\\Windows\\System32\\cmd.exe",
	WSLBash:          "/bin/bash",
	MacDefault:       "/bin/zsh",
	LinuxDefault:     "/bin/bash",
	CSH:              "/bin/csh",
	BASH:             "/bin/bash",
	KSH:              "/bin/ksh",
	SH:               "/bin/sh",
	ZSH:              "/bin/zsh",
	DASH:             "/bin/dash",
	TCSH:             "/bin/tcsh",
	Fallback:         "/bin/sh",
}

// getShellFromEnv gets the shell from environment variables
func getShellFromEnv() string {
	// Load .env file (if exists)
	godotenv.Load()

	if runtime.GOOS == "windows" {
		// On Windows, COMSPEC usually stores cmd.exe
		if comspec := os.Getenv("COMSPEC"); comspec != "" {
			return comspec
		}
		return "C:\\Windows\\System32\\cmd.exe"
	}

	if runtime.GOOS == "darwin" {
		// On macOS, SHELL is a common environment variable
		if shell := os.Getenv("SHELL"); shell != "" {
			return shell
		}
		return "/bin/zsh"
	}

	if runtime.GOOS == "linux" {
		// On Linux, SHELL is a common environment variable
		if shell := os.Getenv("SHELL"); shell != "" {
			return shell
		}
		return "/bin/bash"
	}

	return ""
}

// GetShell returns the shell path for the current system
func GetShell() string {
	// Try environment variables
	envShell := getShellFromEnv()
	if envShell != "" {
		return envShell
	}

	// Fall back to defaults
	if runtime.GOOS == "windows" {
		// On Windows, if we got here, we have no configuration, no COMSPEC
		// Use CMD as a last resort
		return ShellPaths.CMD
	}
	// On macOS/Linux, fall back to POSIX shell
	return ShellPaths.Fallback
}
