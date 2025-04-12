# Tools Test Utility

This utility is used for testing and demonstrating NCA core tool functionalities. It provides a simple command-line interface to invoke these tools.

## Building

Build the tool with the following command:

```bash
make toolstest
```

The compiled binary will be generated at `bin/toolstest`.

## Usage

```
toolstest <tool_name> [parameters]
```

Available tools:
- `execute_command` - Execute command line commands
- `read_file` - Read file contents
- `write_file` - Write content to a file
- `replace_in_file` - Replace content in a file
- `search_files` - Search for content in files
- `list_files` - List files in a directory
- `list_definitions` - List code definition names
- `find_files` - Find files matching a pattern
- `fetch_web` - Fetch web content
- `use_mcp_tool` - Call a tool provided by an MCP server
- `access_mcp_resource` - Access a resource provided by an MCP server

## Examples

1. Execute command:
```bash
./bin/toolstest execute_command --command "ls -la"
```

2. Read file:
```bash
./bin/toolstest read_file --path "Makefile"
```

3. Read specific line range of a file:
```bash
./bin/toolstest read_file --path "Makefile" --range "1-10"
```

4. Write to file:
```bash
./bin/toolstest write_file --path "test.txt" --content "Hello World"
```

5. Search in files:
```bash
./bin/toolstest search_files --path "." --regex "func" --file_pattern "*.go"
```

6. List files in a directory:
```bash
./bin/toolstest list_files --path "." --recursive
```

7. Find files:
```bash
./bin/toolstest find_files --path "." --file_pattern "*.go"
```

8. Use an MCP tool:
```bash
./bin/toolstest use_mcp_tool --server_name "openai" --tool_name "dalle3" --arguments '{"prompt":"a cat"}'
```

9. Access an MCP resource:
```bash
./bin/toolstest access_mcp_resource --server_name "openai" --uri "/resources/images/latest"
```

## Using JSON Parameters

For complex parameters, you can use JSON format via the `--json` parameter:

```bash
./bin/toolstest search_files --json '{"path":".", "regex":"func", "file_pattern":"*.go"}'
```

## Replacing File Content

The `replace_in_file` tool uses a special SEARCH/REPLACE block format:

```bash
./bin/toolstest replace_in_file --path "test.txt" --diff "<<<<<<< SEARCH
original content
=======
new content
>>>>>>> REPLACE"
```

This will search for "original content" in the file and replace it with "new content". 
