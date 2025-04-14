package client

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/pederhe/nca/pkg/mcp/common"
	"github.com/stretchr/testify/assert"
)

func TestStdioTransport(t *testing.T) {
	// Create a test echo server command
	echoCmd := exec.Command("echo", "test")

	params := StdioServerParameters{
		Command: echoCmd.Path,
		Args:    echoCmd.Args[1:],
		Env:     GetDefaultEnvironment(),
	}

	transport := NewStdioClientTransport(params)

	// Test Start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, transport.isConnected)

	// Test Send
	msg := common.JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "test",
		"params":  map[string]interface{}{"key": "value"},
	}

	err = transport.Send(msg)
	assert.NoError(t, err)

	// Test Close
	err = transport.Close()
	assert.NoError(t, err)
	assert.False(t, transport.isConnected)
}

func TestStdioTransportDefaultEnv(t *testing.T) {
	env := GetDefaultEnvironment()

	// Check if essential environment variables are present
	assert.Contains(t, env, "PATH")
	assert.Contains(t, env, "HOME")

	// Check if empty environment variables are not included
	for _, key := range DefaultInheritedEnvVars {
		if os.Getenv(key) == "" {
			assert.NotContains(t, env, key)
		}
	}
}

func TestStdioTransportWithCustomEnv(t *testing.T) {
	customEnv := map[string]string{
		"TEST_VAR": "test_value",
	}

	params := StdioServerParameters{
		Command: "echo",
		Args:    []string{"test"},
		Env:     customEnv,
	}

	transport := NewStdioClientTransport(params)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	assert.NoError(t, err)

	// Verify custom environment variable is set
	assert.Equal(t, customEnv, transport.serverParams.Env)

	err = transport.Close()
	assert.NoError(t, err)
}
