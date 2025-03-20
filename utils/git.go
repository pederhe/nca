package utils

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// GitStatus returns the status of the git repository
func GitStatus() (string, error) {
	cmd := exec.Command("git", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}
	return string(output), nil
}

// GitDiff returns the diff of the git repository
func GitDiff(files []string) (string, error) {
	args := []string{"diff", "--cached"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}
	return string(output), nil
}

// GitAdd adds files to the git staging area
func GitAdd(files []string) error {
	args := []string{"add"}
	if len(files) == 0 {
		args = append(args, "-A")
	} else {
		args = append(args, "--")
		args = append(args, files...)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add files to git: %w\n%s", err, string(output))
	}
	return nil
}

// GitCommit commits the staged changes with the given message
func GitCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w\n%s", err, string(output))
	}
	return nil
}

// GetModifiedFiles returns a list of modified files in the git repository
func GetModifiedFiles() ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get modified files: %w", err)
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 3 {
			// Extract the file path from the status line
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}

	return files, nil
}
