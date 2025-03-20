package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TreeNode represents a node in the directory tree
type TreeNode struct {
	Name     string
	IsDir    bool
	Size     int64
	Children []*TreeNode
}

// DirectoryTree generates a directory tree structure
func DirectoryTree(rootPath string, maxDepth int) (*TreeNode, error) {
	info, err := os.Stat(rootPath)
	if err != nil {
		return nil, err
	}

	root := &TreeNode{
		Name:     filepath.Base(rootPath),
		IsDir:    info.IsDir(),
		Size:     info.Size(),
		Children: []*TreeNode{},
	}

	if !info.IsDir() {
		return root, nil
	}

	err = buildTree(rootPath, root, 0, maxDepth)
	if err != nil {
		return nil, err
	}

	return root, nil
}

// buildTree recursively builds the directory tree
func buildTree(path string, node *TreeNode, currentDepth, maxDepth int) error {
	if maxDepth > 0 && currentDepth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Sort: directories first, then files, sorted alphabetically by name
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		// Skip hidden files and directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		childNode := &TreeNode{
			Name:     entry.Name(),
			IsDir:    entry.IsDir(),
			Size:     info.Size(),
			Children: []*TreeNode{},
		}

		node.Children = append(node.Children, childNode)

		if entry.IsDir() {
			childPath := filepath.Join(path, entry.Name())
			err = buildTree(childPath, childNode, currentDepth+1, maxDepth)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// PrintDirectoryTree prints the directory in a tree structure
func PrintDirectoryTree(root *TreeNode) string {
	var builder strings.Builder
	builder.WriteString(root.Name)
	builder.WriteString("\n")
	printTreeNode(root, "", &builder, true)
	return builder.String()
}

// printTreeNode recursively prints tree nodes
func printTreeNode(node *TreeNode, prefix string, builder *strings.Builder, isRoot bool) {
	if isRoot {
		// Root node has already been printed
		for i, child := range node.Children {
			isLast := i == len(node.Children)-1
			printChild(child, prefix, builder, isLast)
		}
		return
	}

	for i, child := range node.Children {
		isLast := i == len(node.Children)-1
		printChild(child, prefix, builder, isLast)
	}
}

// printChild prints a child node
func printChild(node *TreeNode, prefix string, builder *strings.Builder, isLast bool) {
	// Determine the current line prefix
	if isLast {
		builder.WriteString(prefix + "└── ")
	} else {
		builder.WriteString(prefix + "├── ")
	}

	// Print node name
	if node.IsDir {
		builder.WriteString(node.Name + "/\n")
	} else {
		size := formatSize(node.Size)
		if size != "" {
			builder.WriteString(fmt.Sprintf("%s (%s)\n", node.Name, size))
		} else {
			builder.WriteString(node.Name + "\n")
		}
	}

	// Determine prefix for child nodes
	var newPrefix string
	if isLast {
		newPrefix = prefix + "    "
	} else {
		newPrefix = prefix + "│   "
	}

	// Recursively print child nodes
	printTreeNode(node, newPrefix, builder, false)
}

// formatSize formats file size
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1fGB", float64(size)/(1024*1024*1024))
	}
}

// GetDirectoryTree gets the string representation of the directory tree for a specified path
func GetDirectoryTree(path string, maxDepth int) (string, error) {
	root, err := DirectoryTree(path, maxDepth)
	if err != nil {
		return "", err
	}
	return PrintDirectoryTree(root), nil
}

// GetCurrentDirectoryTree gets the tree structure of the current directory
func GetCurrentDirectoryTree(maxDepth int) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return GetDirectoryTree(currentDir, maxDepth)
}
