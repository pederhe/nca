package core

import (
	"os"
	"testing"
)

func TestCheckpointManager(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Create new CheckpointManager
	cm := NewCheckpointManager()

	// Test checkpoint creation
	t.Run("CreateCheckpoint", func(t *testing.T) {
		cm.CreateCheckpoint("test checkpoint")
		if len(cm.Checkpoints) != 1 {
			t.Errorf("Expected 1 checkpoint, got %d", len(cm.Checkpoints))
		}
		if cm.CurrentCheckpoint == nil {
			t.Error("CurrentCheckpoint should not be nil")
		}
		if cm.CurrentCheckpoint.UserPrompt != "test checkpoint" {
			t.Errorf("Expected user prompt 'test checkpoint', got '%s'", cm.CurrentCheckpoint.UserPrompt)
		}
	})

	// Test recording file operations
	t.Run("RecordFileOperation", func(t *testing.T) {
		// Create test file
		testFile := "test.txt"
		err := os.WriteFile(testFile, []byte("original content"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Record file operation
		cm.RecordFileOperation("write", testFile, "new content", "original content")
		if len(cm.CurrentCheckpoint.Operations) != 1 {
			t.Errorf("Expected 1 operation, got %d", len(cm.CurrentCheckpoint.Operations))
		}
	})

	// Test checkpoint restoration
	t.Run("RestoreCheckpoint", func(t *testing.T) {
		// Modify file content
		testFile := "test.txt"
		err := os.WriteFile(testFile, []byte("modified content"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Restore checkpoint
		result := cm.RestoreCheckpoint(cm.Checkpoints[0].ID)
		if result != "Checkpoint '"+cm.Checkpoints[0].ID+"' successfully restored" {
			t.Errorf("Unexpected restore result: %s", result)
		}

		// Verify file content is restored
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "original content" {
			t.Errorf("Expected content 'original content', got '%s'", string(content))
		}
	})

	// Test checkpoint redo
	t.Run("RedoCheckpoint", func(t *testing.T) {
		// Redo checkpoint
		result := cm.RedoCheckpoint(cm.Checkpoints[0].ID)
		if result != "Operations from checkpoint '"+cm.Checkpoints[0].ID+"' successfully redone" {
			t.Errorf("Unexpected redo result: %s", result)
		}

		// Verify file content is redone
		content, err := os.ReadFile("test.txt")
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "new content" {
			t.Errorf("Expected content 'new content', got '%s'", string(content))
		}
	})

	// Test checkpoint limit
	t.Run("CheckpointLimit", func(t *testing.T) {
		// Create multiple checkpoints
		for i := 0; i < 8; i++ {
			cm.CreateCheckpoint("test checkpoint")
		}

		if len(cm.Checkpoints) > 6 {
			t.Errorf("Expected maximum 6 checkpoints, got %d", len(cm.Checkpoints))
		}
	})

	// Test saving and loading checkpoints
	t.Run("SaveAndLoadCheckpoints", func(t *testing.T) {
		// Save checkpoints
		err := cm.SaveCheckpoints()
		if err != nil {
			t.Fatal(err)
		}

		// Create new CheckpointManager and load checkpoints
		newCM := NewCheckpointManager()
		err = newCM.LoadCheckpoints()
		if err != nil {
			t.Fatal(err)
		}

		// Verify checkpoint count is the same
		if len(newCM.Checkpoints) != len(cm.Checkpoints) {
			t.Errorf("Expected %d checkpoints, got %d", len(cm.Checkpoints), len(newCM.Checkpoints))
		}
	})
}

func TestFileOperations(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	cm := NewCheckpointManager()
	cm.CreateCheckpoint("test operations")

	// Test write operation
	t.Run("WriteOperation", func(t *testing.T) {
		testFile := "write_test.txt"
		cm.RecordFileOperation("write", testFile, "test content", "")
		err := cm.redoFileOperation(cm.CurrentCheckpoint.Operations[0])
		if err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "test content" {
			t.Errorf("Expected content 'test content', got '%s'", string(content))
		}
	})

	// Test replace operation
	t.Run("ReplaceOperation", func(t *testing.T) {
		testFile := "replace_test.txt"
		err := os.WriteFile(testFile, []byte("original"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		cm.RecordFileOperation("replace", testFile, "replaced", "original")
		err = cm.redoFileOperation(cm.CurrentCheckpoint.Operations[1])
		if err != nil {
			t.Fatal(err)
		}

		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "replaced" {
			t.Errorf("Expected content 'replaced', got '%s'", string(content))
		}
	})

	// Test delete operation
	t.Run("DeleteOperation", func(t *testing.T) {
		testFile := "delete_test.txt"
		err := os.WriteFile(testFile, []byte("to delete"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		cm.RecordFileOperation("delete", testFile, "to delete", "")
		err = cm.redoFileOperation(cm.CurrentCheckpoint.Operations[2])
		if err != nil {
			t.Fatal(err)
		}

		_, err = os.Stat(testFile)
		if !os.IsNotExist(err) {
			t.Error("File should have been deleted")
		}
	})
}
