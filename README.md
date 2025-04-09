# NCA (Nano Code Agent)

## Overview

NCA is a lightweight command-line code assistant tool developed in Go, designed to help developers improve coding efficiency. By connecting to various large language model APIs (such as Doubao, Qwen, DeepSeek, etc.), NCA can understand natural language instructions, perform code analysis and generation, and assist in completing various development tasks.

## Key Features

- **Interactive Command Line Interface**: Provides a friendly REPL interface, supporting both one-time queries and continuous dialogue mode
- **Multiple LLM Support**: Supports various large language model API providers, including Doubao, Qwen, and DeepSeek
- **MCP Protocol Support**: Implements the Model Control Protocol (MCP) for standardized communication with language model servers
- **Session Management**: Supports creating checkpoints and saving conversation context
- **Flexible Configuration System**: Supports both local and global configuration items
- **Pipe Input Support**: Can receive input content through pipes
- **Debug Mode**: Provides detailed logging for troubleshooting

## Installation

### Building from Source

```bash
# Clone the repository
git clone https://github.com/pederhe/nca.git
cd nca

# Build the project
make build

# Install to system path
make install
```

## Usage

### Configuring API Providers

```bash
# Set API provider
nca config set model deepseek-chat
nca config set api_key your_api_key_here
```

### MCP Server Configuration

NCA supports MCP servers through a configuration file. Create a `mcp_settings.json` file with the following structure:

```json
{
  "mcp_servers": {
    "local-server": {
      "transportType": "stdio",
      "command": "/path/to/mcp-server",
      "timeout": 60
    },
    "remote-server": {
      "transportType": "sse",
      "url": "https://api.example.com/mcp/events",
      "timeout": 60
    }
  }
}
```

For detailed configuration options, see [MCP Server Configuration](core/mcp/hub/README.md).

### Basic Usage

```bash
# Start interactive mode
nca

# Start with specific instructions
nca "Create a snake game"

# One-time query
nca -p "How to implement a simple HTTP server?"

# Pass input through pipe
cat main.go | nca "Analyze the performance issues in this code"
```

### More Commands

```bash
nca help
```
