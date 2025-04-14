package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pederhe/nca/internal/core"
)

// Help text for the command
const helpText = `
Tools Test CLI - For testing NCA core tool functionalities

Usage:
  toolstest <tool_name> [parameters]

Available tools:
  execute_command     - Execute command line commands
  read_file           - Read file contents
  write_file          - Write content to a file
  replace_in_file     - Replace content in a file
  search_files        - Search for content in files
  list_files          - List files in a directory
  list_definitions    - List code definition names
  find_files          - Find files matching a pattern
  fetch_web           - Fetch web content
  use_mcp_tool        - Call a tool provided by an MCP server
  access_mcp_resource - Access a resource provided by an MCP server

Examples:
  toolstest execute_command --command "ls -la"
  toolstest read_file --path "file.txt" --range "1-10"
  toolstest write_file --path "new.txt" --content "Hello World"
  toolstest search_files --path "." --regex "function"
  toolstest list_files --path "." --recursive
  toolstest use_mcp_tool --server_name "openai" --tool_name "dalle3" --arguments '{"prompt":"cat"}'
`

// ToolFunc represents a tool function with its parameters
type ToolFunc struct {
	Func       func(map[string]interface{}) string
	ParamFlags map[string]*string
	BoolFlags  map[string]*bool
}

func main() {
	// Define available tools with their functions and parameters
	tools := map[string]ToolFunc{
		"execute_command": {
			Func: core.ExecuteCommand,
			ParamFlags: map[string]*string{
				"command": nil,
			},
			BoolFlags: map[string]*bool{
				"requires_approval": nil,
			},
		},
		"read_file": {
			Func: core.ReadFile,
			ParamFlags: map[string]*string{
				"path":  nil,
				"range": nil,
			},
		},
		"write_file": {
			Func: core.WriteToFile,
			ParamFlags: map[string]*string{
				"path":    nil,
				"content": nil,
			},
		},
		"replace_in_file": {
			Func: core.ReplaceInFile,
			ParamFlags: map[string]*string{
				"path": nil,
				"diff": nil,
			},
		},
		"search_files": {
			Func: core.SearchFiles,
			ParamFlags: map[string]*string{
				"path":         nil,
				"regex":        nil,
				"file_pattern": nil,
			},
		},
		"list_files": {
			Func: core.ListFiles,
			ParamFlags: map[string]*string{
				"path": nil,
			},
			BoolFlags: map[string]*bool{
				"recursive": nil,
			},
		},
		"list_definitions": {
			Func: core.ListCodeDefinitionNames,
			ParamFlags: map[string]*string{
				"path": nil,
			},
		},
		"find_files": {
			Func: core.FindFiles,
			ParamFlags: map[string]*string{
				"path":         nil,
				"file_pattern": nil,
			},
		},
		"fetch_web": {
			Func: core.FetchWebContent,
			ParamFlags: map[string]*string{
				"url": nil,
			},
		},
		"use_mcp_tool": {
			Func: core.UseMcpTool,
			ParamFlags: map[string]*string{
				"server_name": nil,
				"tool_name":   nil,
				"arguments":   nil,
			},
		},
		"access_mcp_resource": {
			Func: core.AccessMcpResource,
			ParamFlags: map[string]*string{
				"server_name": nil,
				"uri":         nil,
			},
		},
	}

	// Check if tool name is provided
	if len(os.Args) < 2 {
		fmt.Print(helpText)
		os.Exit(1)
	}

	// Get tool name from command line
	toolName := os.Args[1]

	// Show help if requested or if tool not found
	if toolName == "help" || toolName == "--help" || toolName == "-h" {
		fmt.Print(helpText)
		os.Exit(0)
	}

	tool, exists := tools[toolName]
	if !exists {
		fmt.Printf("Error: Unknown tool name '%s'\n", toolName)
		fmt.Print(helpText)
		os.Exit(1)
	}

	// Create new flag set for the tool
	fs := flag.NewFlagSet(toolName, flag.ExitOnError)

	// Initialize parameter flags for the tool
	for param := range tool.ParamFlags {
		tool.ParamFlags[param] = fs.String(param, "", "Parameter "+param)
	}

	// Initialize boolean flags for the tool
	if tool.BoolFlags != nil {
		for param := range tool.BoolFlags {
			tool.BoolFlags[param] = fs.Bool(param, false, "Boolean "+param)
		}
	}

	// Add JSON parameter flag for passing all parameters as a JSON string
	jsonParams := fs.String("json", "", "JSON string containing all parameters")

	// Parse flags from command line
	err := fs.Parse(os.Args[2:])
	if err != nil {
		fmt.Printf("Error: Failed to parse parameters: %v\n", err)
		os.Exit(1)
	}

	// Prepare parameters for the tool
	params := make(map[string]interface{})

	// If JSON parameter is provided, use it
	if *jsonParams != "" {
		err := json.Unmarshal([]byte(*jsonParams), &params)
		if err != nil {
			fmt.Printf("Error: Failed to parse JSON parameters: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Otherwise, use individual flags
		for param, value := range tool.ParamFlags {
			if value != nil && *value != "" {
				params[param] = *value
			}
		}

		// Add boolean flags
		if tool.BoolFlags != nil {
			for param, value := range tool.BoolFlags {
				if value != nil && *value {
					params[param] = true
				}
			}
		}
	}

	// Check if required parameters are provided
	if (toolName == "execute_command" && params["command"] == nil) ||
		(toolName == "read_file" && params["path"] == nil) ||
		(toolName == "write_file" && (params["path"] == nil || params["content"] == nil)) ||
		(toolName == "replace_in_file" && (params["path"] == nil || params["diff"] == nil)) ||
		(toolName == "search_files" && (params["path"] == nil || params["regex"] == nil)) ||
		(toolName == "list_files" && params["path"] == nil) ||
		(toolName == "list_definitions" && params["path"] == nil) ||
		(toolName == "find_files" && (params["path"] == nil || params["file_pattern"] == nil)) ||
		(toolName == "fetch_web" && params["url"] == nil) ||
		(toolName == "use_mcp_tool" && (params["server_name"] == nil || params["tool_name"] == nil || params["arguments"] == nil)) ||
		(toolName == "access_mcp_resource" && (params["server_name"] == nil || params["uri"] == nil)) {
		fmt.Println("Error: Missing required parameters")
		fmt.Printf("Required parameters: %s\n", strings.Join(getRequiredParams(toolName), ", "))
		os.Exit(1)
	}

	// Execute the tool function with the parameters
	result := tool.Func(params)

	// Print the result
	fmt.Println(result)
}

// getRequiredParams returns the required parameters for a tool
func getRequiredParams(toolName string) []string {
	switch toolName {
	case "execute_command":
		return []string{"command"}
	case "read_file":
		return []string{"path"}
	case "write_file":
		return []string{"path", "content"}
	case "replace_in_file":
		return []string{"path", "diff"}
	case "search_files":
		return []string{"path", "regex"}
	case "list_files":
		return []string{"path"}
	case "list_definitions":
		return []string{"path"}
	case "find_files":
		return []string{"path", "file_pattern"}
	case "fetch_web":
		return []string{"url"}
	case "use_mcp_tool":
		return []string{"server_name", "tool_name", "arguments"}
	case "access_mcp_resource":
		return []string{"server_name", "uri"}
	default:
		return []string{}
	}
}
