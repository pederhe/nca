package common

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestReadBuffer_ReadMessage(t *testing.T) {
	rb := NewReadBuffer()

	// Test empty buffer
	message, err := rb.ReadMessage()
	if err != nil {
		t.Errorf("Expected no error for empty buffer, got: %v", err)
	}
	if message != nil {
		t.Errorf("Expected nil message for empty buffer, got: %v", message)
	}

	// Test single complete message
	jsonMsg := `{"jsonrpc":"2.0","id":1,"method":"test"}`
	rb.Append([]byte(jsonMsg + "\n"))

	message, err = rb.ReadMessage()
	if err != nil {
		t.Errorf("Expected no error for valid message, got: %v", err)
	}

	expected := JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      float64(1), // JSON numbers are parsed as float64
		"method":  "test",
	}

	if !reflect.DeepEqual(message, expected) {
		t.Errorf("Expected message to be %v, got %v", expected, message)
	}

	// Test no messages left
	message, err = rb.ReadMessage()
	if err != nil {
		t.Errorf("Expected no error for depleted buffer, got: %v", err)
	}
	if message != nil {
		t.Errorf("Expected nil message for depleted buffer, got: %v", message)
	}

	// Test message split into two parts
	rb.Clear()
	rb.Append([]byte(`{"jsonrpc":"2.0","id":2,"method":"test2"}`))
	rb.Append([]byte("\n"))

	message, err = rb.ReadMessage()
	if err != nil {
		t.Errorf("Expected no error for completed message, got: %v", err)
	}

	expected = JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      float64(2),
		"method":  "test2",
	}

	if !reflect.DeepEqual(message, expected) {
		t.Errorf("Expected message to be %v, got %v", expected, message)
	}

	// Test multiple messages
	rb.Clear()
	multiMsg := `{"jsonrpc":"2.0","id":3,"method":"test3"}
{"jsonrpc":"2.0","id":4,"method":"test4"}
`
	rb.Append([]byte(multiMsg))

	// Read first message
	message, err = rb.ReadMessage()
	if err != nil {
		t.Errorf("Expected no error for first message, got: %v", err)
	}

	expected = JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      float64(3),
		"method":  "test3",
	}

	if !reflect.DeepEqual(message, expected) {
		t.Errorf("Expected first message to be %v, got %v", expected, message)
	}

	// Read second message
	message, err = rb.ReadMessage()
	if err != nil {
		t.Errorf("Expected no error for second message, got: %v", err)
	}

	expected = JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      float64(4),
		"method":  "test4",
	}

	if !reflect.DeepEqual(message, expected) {
		t.Errorf("Expected second message to be %v, got %v", expected, message)
	}
}

func TestReadBuffer_Clear(t *testing.T) {
	rb := NewReadBuffer()

	// Add some data to the buffer
	rb.Append([]byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`))

	// Clear the buffer
	rb.Clear()

	// Check if buffer is empty
	message, err := rb.ReadMessage()
	if err != nil {
		t.Errorf("Expected no error after clear, got: %v", err)
	}
	if message != nil {
		t.Errorf("Expected nil message after clear, got: %v", message)
	}
}

func TestDeserializeMessage(t *testing.T) {
	// Test valid message
	validJson := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	message, err := DeserializeMessage(validJson)
	if err != nil {
		t.Errorf("Expected no error for valid JSON, got: %v", err)
	}

	expected := JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      float64(1),
		"method":  "test",
	}

	if !reflect.DeepEqual(message, expected) {
		t.Errorf("Expected message to be %v, got %v", expected, message)
	}

	// Test notification (no id, only method)
	notificationJson := []byte(`{"jsonrpc":"2.0","method":"notify"}`)
	message, err = DeserializeMessage(notificationJson)
	if err != nil {
		t.Errorf("Expected no error for notification JSON, got: %v", err)
	}

	expected = JSONRPCMessage{
		"jsonrpc": "2.0",
		"method":  "notify",
	}

	if !reflect.DeepEqual(message, expected) {
		t.Errorf("Expected message to be %v, got %v", expected, message)
	}

	// Test invalid JSON
	invalidJson := []byte(`{not-valid-json}`)
	message, err = DeserializeMessage(invalidJson)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if message != nil {
		t.Errorf("Expected nil message for invalid JSON, got: %v", message)
	}

	// Test invalid message format (missing both id and method)
	invalidMsgJson := []byte(`{"jsonrpc":"2.0","params":{}}`)
	message, err = DeserializeMessage(invalidMsgJson)
	if err == nil {
		t.Error("Expected error for invalid message format, got nil")
	}
	if message != nil {
		t.Errorf("Expected nil message for invalid message format, got: %v", message)
	}
}

func TestSerializeMessage(t *testing.T) {
	// Test request message
	requestMsg := JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "test",
		"params":  map[string]interface{}{"name": "value"},
	}

	bytes, err := SerializeMessage(requestMsg)
	if err != nil {
		t.Errorf("Expected no error for valid message, got: %v", err)
	}

	// Deserialize to verify
	message, err := DeserializeMessage(bytes[:len(bytes)-1]) // Remove trailing newline
	if err != nil {
		t.Errorf("Expected no error when deserializing serialized message, got: %v", err)
	}

	// Note: The numeric types might be different after serialization/deserialization
	requestMsg["id"] = float64(1)

	if !reflect.DeepEqual(message, requestMsg) {
		t.Errorf("Expected message to be %v, got %v", requestMsg, message)
	}

	// Check for newline at the end
	if bytes[len(bytes)-1] != '\n' {
		t.Error("Expected serialized message to end with newline")
	}
}

func TestReadMessages(t *testing.T) {
	// Create a reader with multiple messages
	input := `{"jsonrpc":"2.0","id":1,"method":"test1"}
{"jsonrpc":"2.0","id":2,"method":"test2"}
{"jsonrpc":"2.0","id":3,"method":"test3"}
`
	reader := strings.NewReader(input)

	// Collect messages
	var messages []JSONRPCMessage
	err := ReadMessages(reader, func(message JSONRPCMessage) error {
		messages = append(messages, message)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error from ReadMessages, got: %v", err)
	}

	// Check number of messages
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Check content of messages
	expected := []JSONRPCMessage{
		{
			"jsonrpc": "2.0",
			"id":      float64(1),
			"method":  "test1",
		},
		{
			"jsonrpc": "2.0",
			"id":      float64(2),
			"method":  "test2",
		},
		{
			"jsonrpc": "2.0",
			"id":      float64(3),
			"method":  "test3",
		},
	}

	for i, msg := range messages {
		if !reflect.DeepEqual(msg, expected[i]) {
			t.Errorf("Expected message %d to be %v, got %v", i, expected[i], msg)
		}
	}
}

func TestWriteMessage(t *testing.T) {
	// Create message
	message := JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "test",
	}

	// Create buffer to write to
	var buffer bytes.Buffer

	// Write message
	err := WriteMessage(&buffer, message)
	if err != nil {
		t.Errorf("Expected no error from WriteMessage, got: %v", err)
	}

	// Check buffer content
	bytes := buffer.Bytes()
	if len(bytes) == 0 {
		t.Error("Expected non-empty buffer after WriteMessage")
	}

	// Deserialize to verify
	deserializedMsg, err := DeserializeMessage(bytes[:len(bytes)-1]) // Remove trailing newline
	if err != nil {
		t.Errorf("Expected no error when deserializing written message, got: %v", err)
	}

	expected := JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      float64(1), // JSON numbers are parsed as float64
		"method":  "test",
	}

	if !reflect.DeepEqual(deserializedMsg, expected) {
		t.Errorf("Expected message to be %v, got %v", expected, deserializedMsg)
	}

	// Check for newline at the end
	if bytes[len(bytes)-1] != '\n' {
		t.Error("Expected written message to end with newline")
	}
}
