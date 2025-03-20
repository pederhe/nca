package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileOperation represents a file operation that can be undone/redone
type FileOperation struct {
	Type       string    // "write", "replace", "delete"
	Path       string    // File path
	Content    string    // For write/replace operations: new content
	OldContent string    // For replace operations: previous content
	Timestamp  time.Time // When the operation occurred
}

// Checkpoint represents a saved state that can be restored
type Checkpoint struct {
	ID         string          // Unique identifier for the checkpoint
	UserPrompt string          // The user prompt that initiated this checkpoint
	Timestamp  time.Time       // When the checkpoint was created
	Operations []FileOperation // Operations performed after this checkpoint
}

// CheckpointManager manages checkpoints
type CheckpointManager struct {
	Checkpoints       []Checkpoint // List of all checkpoints
	CurrentCheckpoint *Checkpoint  // Current checkpoint being recorded
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager() *CheckpointManager {
	return &CheckpointManager{
		Checkpoints:       []Checkpoint{},
		CurrentCheckpoint: nil,
	}
}

// RecordFileOperation records a file operation for potential undo/redo
func (cm *CheckpointManager) RecordFileOperation(operationType string, path string, content string, oldContent string) {
	if cm.CurrentCheckpoint == nil {
		return // No active checkpoint to record to
	}

	operation := FileOperation{
		Type:       operationType,
		Path:       path,
		Content:    content,
		OldContent: oldContent,
		Timestamp:  time.Now(),
	}

	cm.CurrentCheckpoint.Operations = append(cm.CurrentCheckpoint.Operations, operation)
}

// CreateCheckpoint creates a new checkpoint with the given user prompt
func (cm *CheckpointManager) CreateCheckpoint(userPrompt string) {
	// Generate a unique ID based on timestamp
	id := time.Now().Format("20060102-150405")

	// Create a new checkpoint
	checkpoint := Checkpoint{
		ID:         id,
		UserPrompt: userPrompt,
		Timestamp:  time.Now(),
		Operations: []FileOperation{},
	}

	// Add to the list of checkpoints
	cm.Checkpoints = append(cm.Checkpoints, checkpoint)

	// Limit the number of checkpoints to 6
	if len(cm.Checkpoints) > 6 {
		// Keep only the latest 6 checkpoints
		cm.Checkpoints = cm.Checkpoints[len(cm.Checkpoints)-6:]
	}

	// Set as current checkpoint
	cm.CurrentCheckpoint = &cm.Checkpoints[len(cm.Checkpoints)-1]

	// Save checkpoints to file
	if err := cm.SaveCheckpoints(); err != nil {
		fmt.Printf("Warning: Failed to save checkpoints after creating new checkpoint: %s\n", err)
	}
}

// ListCheckpoints returns formatted information about all checkpoints
func (cm *CheckpointManager) ListCheckpoints() string {
	if len(cm.Checkpoints) == 0 {
		return "No checkpoints available."
	}

	var result strings.Builder
	result.WriteString("Available checkpoints:\n")
	result.WriteString("checkpoint_id        user_prompt                        time\n")
	result.WriteString("----------------------------------------------------------------\n")

	for _, cp := range cm.Checkpoints {
		// Truncate user prompt if it's too long
		prompt := cp.UserPrompt
		if len(prompt) > 30 {
			prompt = prompt[:27] + "..."
		}

		// Format timestamp
		timeStr := cp.Timestamp.Format("2006-01-02 15:04:05")

		// Format line with fixed width columns
		result.WriteString(fmt.Sprintf("%-20s %-35s %s\n", cp.ID, prompt, timeStr))
	}

	return result.String()
}

// undoFileOperation undoes a single file operation
func (cm *CheckpointManager) undoFileOperation(op FileOperation) error {
	switch op.Type {
	case "write":
		// For a write operation, delete the file if it didn't exist before
		if op.OldContent == "" {
			return os.Remove(op.Path)
		}
		// Otherwise restore the previous content
		return os.WriteFile(op.Path, []byte(op.OldContent), 0644)

	case "replace":
		// For a replace operation, restore the previous content
		return os.WriteFile(op.Path, []byte(op.OldContent), 0644)

	case "delete":
		// For a delete operation, recreate the file with its previous content
		// Ensure directory exists
		dir := filepath.Dir(op.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return os.WriteFile(op.Path, []byte(op.Content), 0644)
	}

	return fmt.Errorf("unknown operation type: %s", op.Type)
}

// redoFileOperation redoes a single file operation
func (cm *CheckpointManager) redoFileOperation(op FileOperation) error {
	switch op.Type {
	case "write", "replace":
		// Ensure directory exists
		dir := filepath.Dir(op.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return os.WriteFile(op.Path, []byte(op.Content), 0644)

	case "delete":
		return os.Remove(op.Path)
	}

	return fmt.Errorf("unknown operation type: %s", op.Type)
}

// RestoreCheckpoint undoes all operations back to the specified checkpoint
func (cm *CheckpointManager) RestoreCheckpoint(checkpointID string) string {
	// Find the checkpoint
	var targetIndex = -1
	for i, cp := range cm.Checkpoints {
		if cp.ID == checkpointID {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return fmt.Sprintf("Error: Checkpoint '%s' not found", checkpointID)
	}

	// Undo operations in reverse order, starting from the most recent checkpoint
	var errors []string
	for i := len(cm.Checkpoints) - 1; i >= targetIndex; i-- {
		cp := cm.Checkpoints[i]

		// Undo each operation in this checkpoint in reverse order
		for j := len(cp.Operations) - 1; j >= 0; j-- {
			op := cp.Operations[j]
			if err := cm.undoFileOperation(op); err != nil {
				errors = append(errors, fmt.Sprintf("Error undoing %s operation on %s: %s", op.Type, op.Path, err))
			}
		}
	}

	// Set current checkpoint
	if len(cm.Checkpoints) > 0 {
		cm.CurrentCheckpoint = &cm.Checkpoints[len(cm.Checkpoints)-1]
	} else {
		cm.CurrentCheckpoint = nil
	}

	if len(errors) > 0 {
		return fmt.Sprintf("Checkpoint partially restored with errors:\n%s", strings.Join(errors, "\n"))
	}

	return fmt.Sprintf("Checkpoint '%s' successfully restored", checkpointID)
}

// RedoCheckpoint redoes all operations from the specified checkpoint
func (cm *CheckpointManager) RedoCheckpoint(checkpointID string) string {
	// Find the checkpoint
	var targetIndex = -1
	for i, cp := range cm.Checkpoints {
		if cp.ID == checkpointID {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return fmt.Sprintf("Error: Checkpoint '%s' not found", checkpointID)
	}

	// Redo operations in order, starting from the specified checkpoint
	var errors []string
	for i := targetIndex; i < len(cm.Checkpoints); i++ {
		cp := cm.Checkpoints[i]

		// Redo each operation in this checkpoint in order
		for _, op := range cp.Operations {
			if err := cm.redoFileOperation(op); err != nil {
				errors = append(errors, fmt.Sprintf("Error redoing %s operation on %s: %s", op.Type, op.Path, err))
			}
		}
	}

	// Set current checkpoint to the last one
	if len(cm.Checkpoints) > 0 {
		cm.CurrentCheckpoint = &cm.Checkpoints[len(cm.Checkpoints)-1]
	}

	// Save checkpoints after redoing operations
	if err := cm.SaveCheckpoints(); err != nil {
		fmt.Printf("Warning: Failed to save checkpoints after redo: %s\n", err)
	}

	if len(errors) > 0 {
		return fmt.Sprintf("Checkpoint partially redone with errors:\n%s", strings.Join(errors, "\n"))
	}

	return fmt.Sprintf("Operations from checkpoint '%s' successfully redone", checkpointID)
}

// HandleCheckpointCommand handles the /checkpoint command
func (cm *CheckpointManager) HandleCheckpointCommand(args []string) string {
	if len(args) == 0 {
		return "Usage: /checkpoint [list|restore|redo] [checkpoint_id]"
	}

	switch args[0] {
	case "list":
		return cm.ListCheckpoints()

	case "restore":
		if len(args) < 2 {
			return "Usage: /checkpoint restore <checkpoint_id>"
		}
		return cm.RestoreCheckpoint(args[1])

	case "redo":
		if len(args) < 2 {
			return "Usage: /checkpoint redo <checkpoint_id>"
		}
		return cm.RedoCheckpoint(args[1])

	default:
		return fmt.Sprintf("Unknown checkpoint command: %s\nUsage: /checkpoint [list|restore|redo] [checkpoint_id]", args[0])
	}
}

// SaveCheckpoints saves checkpoints to a file
func (cm *CheckpointManager) SaveCheckpoints() error {
	// Save checkpoints in the .nca directory of the working directory, not in the user's home directory
	checkpointDir := filepath.Join(".nca")
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		return err
	}

	// Path to checkpoints file
	checkpointFile := filepath.Join(checkpointDir, "checkpoints.json")

	// Marshal the checkpoints to JSON
	data, err := json.MarshalIndent(cm.Checkpoints, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(checkpointFile, data, 0644)
}

// LoadCheckpoints loads checkpoints from a file
func (cm *CheckpointManager) LoadCheckpoints() error {
	// Load checkpoint data from the .nca directory in the working directory
	checkpointFile := filepath.Join(".nca", "checkpoints.json")

	// Check if file exists
	if _, err := os.Stat(checkpointFile); os.IsNotExist(err) {
		// Initialize empty checkpoints
		cm.Checkpoints = []Checkpoint{}
		return nil
	}

	// Read file
	data, err := os.ReadFile(checkpointFile)
	if err != nil {
		return err
	}

	// Unmarshal the JSON
	return json.Unmarshal(data, &cm.Checkpoints)
}
