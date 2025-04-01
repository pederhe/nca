package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// ReadBuffer buffers a continuous stdio stream into discrete JSON-RPC messages.
type ReadBuffer struct {
	buffer []byte
}

// NewReadBuffer creates a new ReadBuffer.
func NewReadBuffer() *ReadBuffer {
	return &ReadBuffer{}
}

// Append adds a chunk of data to the buffer.
func (rb *ReadBuffer) Append(chunk []byte) {
	if rb.buffer == nil {
		rb.buffer = chunk
	} else {
		rb.buffer = append(rb.buffer, chunk...)
	}
}

// ReadMessage reads a complete JSON-RPC message from the buffer.
// Returns nil if no complete message is available.
func (rb *ReadBuffer) ReadMessage() (JSONRPCMessage, error) {
	if rb.buffer == nil {
		return nil, nil
	}

	index := bytes.IndexByte(rb.buffer, '\n')
	if index == -1 {
		return nil, nil
	}

	line := rb.buffer[:index]
	rb.buffer = rb.buffer[index+1:]

	return DeserializeMessage(line)
}

// Clear empties the buffer.
func (rb *ReadBuffer) Clear() {
	rb.buffer = nil
}

// DeserializeMessage converts a JSON string into a JSONRPCMessage.
func DeserializeMessage(line []byte) (JSONRPCMessage, error) {
	var message JSONRPCMessage
	err := json.Unmarshal(line, &message)
	if err != nil {
		return nil, err
	}

	// Validate that it's a proper JSON-RPC message
	if _, hasID := message["id"]; !hasID && message["method"] == nil {
		return nil, errors.New("invalid JSON-RPC message: missing id or method")
	}

	return message, nil
}

// SerializeMessage converts a JSONRPCMessage to a JSON string with a newline.
func SerializeMessage(message JSONRPCMessage) ([]byte, error) {
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	return append(jsonBytes, '\n'), nil
}

// ReadMessages continuously reads JSON-RPC messages from a reader and passes them to a handler function.
func ReadMessages(reader io.Reader, handler func(JSONRPCMessage) error) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		message, err := DeserializeMessage(line)
		if err != nil {
			return err
		}

		if err := handler(message); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// WriteMessage writes a JSON-RPC message to a writer.
func WriteMessage(writer io.Writer, message JSONRPCMessage) error {
	bytes, err := SerializeMessage(message)
	if err != nil {
		return err
	}

	_, err = writer.Write(bytes)
	return err
}
