package test

import (
	"fmt"
	"testing"

	"github.com/pederhe/nca/utils"
)

func TestDirectoryTree(t *testing.T) {
	// Test the tree structure of the current directory
	tree, err := utils.GetCurrentDirectoryTree(2) // Limit depth to 2 levels
	if err != nil {
		t.Fatalf("Failed to get directory tree: %v", err)
	}

	fmt.Println("Current directory tree structure (depth 2):")
	fmt.Println(tree)

	// Test the tree structure of a specified directory
	tree, err = utils.GetDirectoryTree("../", 1) // Parent directory, depth of 1 level
	if err != nil {
		t.Fatalf("Failed to get specified directory tree: %v", err)
	}

	fmt.Println("\nParent directory tree structure (depth 1):")
	fmt.Println(tree)
}
