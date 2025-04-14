# MCP Server Configuration

This document explains how to configure MCP servers using the settings file.

## Configuration File Format

The MCP server configuration is stored in a JSON file with the following structure:

```json
{
  "mcp_servers": {
    "server-name": {
      "transportType": "stdio|sse",
      "command": "/path/to/command",
      "args": ["arg1", "arg2"],
      "env": {
        "KEY1": "value1"
      },
      "timeout": 60,
      "autoApprove": ["namespace1"],
      "disabled": false
    }
  }
}
```

## Configuration Fields

### Common Fields

- `transportType` (required): The type of transport to use
  - `stdio`: Standard input/output transport
  - `sse`: Server-Sent Events transport
- `timeout` (optional): Connection timeout in seconds
  - Default: 60 seconds
  - Minimum: 10 seconds
- `autoApprove` (optional): List of namespaces to auto-approve
- `disabled` (optional): Whether the server is disabled
  - Default: false

### Stdio Transport Specific Fields

- `command` (required): The command to execute
- `args` (optional): Command line arguments
- `env` (optional): Environment variables

### SSE Transport Specific Fields

- `url` (required): The SSE server URL

## Example Configurations

### Stdio Transport Example

```json
{
  "mcp_servers": {
    "local-server": {
      "transportType": "stdio",
      "command": "/usr/local/bin/mcp-server",
      "args": ["--config", "config.json"],
      "env": {
        "DEBUG": "true"
      },
      "timeout": 60,
      "autoApprove": ["default"],
      "disabled": false
    }
  }
}
```

### SSE Transport Example

```json
{
  "mcp_servers": {
    "remote-server": {
      "transportType": "sse",
      "url": "https://api.example.com/mcp/events",
      "timeout": 60,
      "autoApprove": ["prod", "staging"],
      "disabled": false
    }
  }
}
```

## Multiple Server Configuration

You can configure multiple servers in the same file:

```json
{
  "mcp_servers": {
    "local-server": {
      "transportType": "stdio",
      "command": "/usr/local/bin/mcp-server",
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

## Validation Rules

The configuration is validated according to the following rules:

1. `timeout` must be at least 10 seconds
2. For `stdio` transport:
   - `command` is required
3. For `sse` transport:
   - `url` is required
4. Invalid transport types will be rejected

## Error Handling

If the configuration is invalid, the system will return an error with details about what went wrong. Common errors include:

- Missing required fields
- Invalid timeout values
- Invalid transport types
- Malformed JSON

## Best Practices

1. Always specify a reasonable timeout value
2. Use descriptive server names
3. Keep sensitive information in environment variables
4. Regularly review and update auto-approve lists
5. Disable unused servers instead of removing them 