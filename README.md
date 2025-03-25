# NCA (Nano Code Agent)

## Overview

NCA is a lightweight command-line code assistant tool developed in Go, designed to help developers improve coding efficiency. By connecting to various large language model APIs (such as Doubao, Qwen, DeepSeek, etc.), NCA can understand natural language instructions, perform code analysis and generation, and assist in completing various development tasks.

## Key Features

- **Interactive Command Line Interface**: Provides a friendly REPL interface, supporting both one-time queries and continuous dialogue mode
- **Multiple LLM Support**: Supports various large language model API providers, including Doubao, Qwen, and DeepSeek
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
