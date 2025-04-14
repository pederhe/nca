package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pederhe/nca/pkg/mcp/common"
)

// StdioServerParameters defines parameters for starting a server process
type StdioServerParameters struct {
	// Command is the executable to run
	Command string

	// Args are the command line arguments to pass to the executable
	Args []string

	// Env is the environment variables to use when starting the process
	// If not specified, the result of GetDefaultEnvironment() will be used
	Env map[string]string

	// Stderr specifies how to handle the child process's standard error output
	// Default is "inherit", meaning error messages will be printed to the parent process's standard error
	Stderr io.Writer

	// Cwd specifies the working directory to use when starting the process
	// If not specified, the current working directory will be used
	Cwd string
}

// DefaultInheritedEnvVars is the default environment variables to inherit
var DefaultInheritedEnvVars = []string{
	"HOME", "LOGNAME", "PATH", "SHELL", "TERM", "USER",
}

// GetDefaultEnvironment returns a default environment object containing safe variables to inherit
func GetDefaultEnvironment() map[string]string {
	env := make(map[string]string)

	for _, key := range DefaultInheritedEnvVars {
		value := os.Getenv(key)
		if value == "" {
			continue
		}

		env[key] = value
	}

	return env
}

// StdioClientTransport implements a client transport based on standard input/output
// This will connect to the server by generating a process and communicating with it via stdin/stdout
type StdioClientTransport struct {
	process        *exec.Cmd
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	stderr         io.ReadCloser
	serverParams   StdioServerParameters
	readBuffer     *common.ReadBuffer
	closeHandler   func()
	errorHandler   func(error)
	messageHandler func(common.JSONRPCMessage)
	sessionID      string
	mutex          sync.Mutex
	isConnected    bool
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewStdioClientTransport creates a new standard input/output client transport instance
func NewStdioClientTransport(params StdioServerParameters) *StdioClientTransport {
	return &StdioClientTransport{
		serverParams: params,
		readBuffer:   common.NewReadBuffer(),
	}
}

// Start starts the server process and prepares to communicate with it
func (t *StdioClientTransport) Start(ctx context.Context) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.isConnected {
		return errors.New("StdioClientTransport is already started! If using the Client class, note that Connect() will automatically call Start()")
	}

	t.ctx, t.cancel = context.WithCancel(ctx)

	// Create the command
	t.process = exec.CommandContext(t.ctx, t.serverParams.Command, t.serverParams.Args...)

	// explicitly set process group, so subprocess can receive termination signals
	// on Unix systems, this helps ensure that all processes in the process group can be terminated
	t.process.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Set environment
	if t.serverParams.Env != nil {
		env := make([]string, 0, len(t.serverParams.Env))
		for k, v := range t.serverParams.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		t.process.Env = env
	} else {
		env := GetDefaultEnvironment()
		envVars := make([]string, 0, len(env))
		for k, v := range env {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		t.process.Env = envVars
	}

	// Set working directory
	if t.serverParams.Cwd != "" {
		t.process.Dir = t.serverParams.Cwd
	}

	// Set standard input/output
	var err error
	t.stdin, err = t.process.StdinPipe()
	if err != nil {
		return fmt.Errorf("cannot create stdin pipe: %w", err)
	}

	t.stdout, err = t.process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cannot create stdout pipe: %w", err)
	}

	// Set standard error
	if t.serverParams.Stderr != nil {
		t.process.Stderr = t.serverParams.Stderr
	} else {
		t.stderr, err = t.process.StderrPipe()
		if err != nil {
			return fmt.Errorf("cannot create stderr pipe: %w", err)
		}

		// handle subprocess standard error output
		go func() {
			// If no stderr handler is provided, default it to connect to parent process's stderr
			if t.errorHandler == nil {
				io.Copy(os.Stderr, t.stderr)
			} else {
				// read but not display (discard output), but record critical errors
				scanner := bufio.NewScanner(t.stderr)
				for scanner.Scan() {
					line := scanner.Text()
					// record lines containing "error", "fatal", "panic" to error handler
					if strings.Contains(strings.ToLower(line), "error") ||
						strings.Contains(strings.ToLower(line), "fatal") ||
						strings.Contains(strings.ToLower(line), "panic") {
						if t.errorHandler != nil {
							t.errorHandler(fmt.Errorf("subprocess error: %s", line))
						}
					}
				}
			}
		}()
	}

	// Start the process
	if err := t.process.Start(); err != nil {
		return fmt.Errorf("cannot start process: %w", err)
	}

	t.isConnected = true

	// Start reading stdout
	go t.readStdout()

	// Wait for process to exit and call close handler when done
	go func() {
		err := t.process.Wait()
		t.mutex.Lock()
		t.isConnected = false
		t.mutex.Unlock()

		if err != nil && t.errorHandler != nil && !errors.Is(err, context.Canceled) {
			t.errorHandler(fmt.Errorf("process exited abnormally: %w", err))
		}

		// ensure process has fully terminated
		if t.process != nil && t.process.Process != nil {
			// if process is still running (this should be rare, since we already called Wait)
			if t.process.ProcessState == nil || !t.process.ProcessState.Exited() {
				// force terminate process
				t.process.Process.Kill()
			}
		}

		t.handleClose()
	}()

	return nil
}

// readStdout reads data from standard output and processes JSON-RPC messages
func (t *StdioClientTransport) readStdout() {
	buffer := make([]byte, 4096)

	for {
		n, err := t.stdout.Read(buffer)
		if err != nil {
			if err == io.EOF || errors.Is(err, context.Canceled) || errors.Is(err, os.ErrClosed) {
				return
			}

			if t.errorHandler != nil {
				t.errorHandler(fmt.Errorf("read stdout error: %w", err))
			}
			return
		}

		if n > 0 {
			t.readBuffer.Append(buffer[:n])
			t.processBuffer()
		}
	}
}

// processBuffer processes messages from the read buffer
func (t *StdioClientTransport) processBuffer() {
	for {
		message, err := t.readBuffer.ReadMessage()
		if err != nil {
			if t.errorHandler != nil {
				t.errorHandler(fmt.Errorf("parse message error: %w", err))
			}
			continue
		}

		if message == nil {
			break
		}

		if t.messageHandler != nil {
			t.messageHandler(message)
		}
	}
}

// Close closes the connection and terminates the process
func (t *StdioClientTransport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.isConnected {
		return nil
	}

	// Cancel context, terminate process
	if t.cancel != nil {
		t.cancel()
	}

	// ensure subprocess is terminated
	if t.process != nil && t.process.Process != nil {
		// first try to gracefully close subprocess
		if t.stdin != nil {
			t.stdin.Close()
		}

		// wait a short time to see if process exits by itself
		done := make(chan error, 1)
		go func() {
			done <- t.process.Wait()
		}()

		select {
		case <-done:
			// process has exited, no need to do anything
		case <-time.After(100 * time.Millisecond):
			// if process does not exit by itself, force terminate it
			t.process.Process.Kill()
		}
	}

	t.isConnected = false
	t.readBuffer.Clear()

	return nil
}

// Send sends a JSON-RPC message to the process's standard input
func (t *StdioClientTransport) Send(msg common.JSONRPCMessage) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.isConnected || t.stdin == nil {
		return errors.New("not connected")
	}

	data, err := common.SerializeMessage(msg)
	if err != nil {
		return fmt.Errorf("serialize message error: %w", err)
	}

	_, err = t.stdin.Write(data)
	if err != nil {
		return fmt.Errorf("write to stdin error: %w", err)
	}

	return nil
}

// SetCloseHandler sets the connection close callback
func (t *StdioClientTransport) SetCloseHandler(handler func()) {
	t.closeHandler = handler
}

// SetErrorHandler sets the error handling callback
func (t *StdioClientTransport) SetErrorHandler(handler func(error)) {
	t.errorHandler = handler
}

// SetMessageHandler sets the message reception callback
func (t *StdioClientTransport) SetMessageHandler(handler func(common.JSONRPCMessage)) {
	t.messageHandler = handler
}

// SessionID returns the session ID
func (t *StdioClientTransport) SessionID() string {
	return t.sessionID
}

// Stderr returns the child process's standard error stream
func (t *StdioClientTransport) Stderr() io.ReadCloser {
	return t.stderr
}

// handleClose handles connection close events
func (t *StdioClientTransport) handleClose() {
	// ensure process is terminated
	if t.process != nil && t.process.Process != nil {
		// check if process is still running
		if t.process.ProcessState == nil || !t.process.ProcessState.Exited() {
			// process is still running, force terminate it
			t.process.Process.Kill()
		}
	}

	if t.closeHandler != nil {
		t.closeHandler()
	}
}
