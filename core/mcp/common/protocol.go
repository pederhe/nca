package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DefaultRequestTimeoutMsec is the default request timeout in milliseconds
const DefaultRequestTimeoutMsec = 60000

// RequestID is the unique identifier for a request
type RequestID interface{}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface
func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// ErrorCode defines the JSON-RPC error codes
type ErrorCode int

// Common error codes
const (
	ParseError     ErrorCode = -32700
	InvalidRequest ErrorCode = -32600
	MethodNotFound ErrorCode = -32601
	InvalidParams  ErrorCode = -32602
	InternalError  ErrorCode = -32603
	RequestTimeout ErrorCode = -32000
)

// McpError defines MCP specific errors
type McpError struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface
func (e *McpError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// Progress represents progress notification data
type Progress struct {
	Message string      `json:"message,omitempty"`
	Percent float64     `json:"percent,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ProgressCallback is the type for progress notification callbacks
type ProgressCallback func(progress Progress)

// ProtocolOptions are additional options for protocol initialization
type ProtocolOptions struct {
	// Whether to restrict outgoing requests to only those that the remote party has indicated they can handle
	// Note: This does not affect local capability checks, as incorrectly specifying local capabilities is considered a logical error
	// Currently defaults to false for backward compatibility with SDK versions that don't properly advertise capabilities
	// In the future, this will default to true
	EnforceStrictCapabilities bool
}

// RequestOptions are additional options for each request
type RequestOptions struct {
	// If set, request progress notifications from the remote end (if supported)
	// This callback will be called when progress notifications are received
	OnProgress ProgressCallback

	// Can be used to cancel an in-progress request
	// This will cause AbortError to be thrown from request()
	Signal context.Context

	// Timeout for this request in milliseconds
	// If exceeded, McpError with RequestTimeout error code will be thrown from request()
	// If not specified, DefaultRequestTimeoutMsec will be used as the timeout
	Timeout int

	// If true, receiving progress notifications will reset the request timeout
	// This is useful for long-running operations that send periodic progress updates
	// Default: false
	ResetTimeoutOnProgress bool

	// Maximum total time to wait for a response in milliseconds
	// If exceeded, McpError with RequestTimeout error code will be thrown from request()
	// If not specified, there is no maximum total timeout
	MaxTotalTimeout int
}

// RequestHandlerExtra is additional data provided to request handlers
type RequestHandlerExtra struct {
	// Signal used to notify if the request has been cancelled by the sender
	Signal context.Context

	// Session ID from the transport (if any)
	SessionID string
}

// TimeoutInfo saves request timeout information
type TimeoutInfo struct {
	timer        *time.Timer
	startTime    time.Time
	timeout      int
	maxTotalTime int
	onTimeout    func()
}

// Request represents a MCP protocol request
type Request struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// Notification represents a MCP protocol notification
type Notification struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// Result represents a MCP protocol response result
type Result interface{}

// ServerCapabilities represents server capabilities
type ServerCapabilities struct {
	// Additional server capabilities can be added here
}

// ClientCapabilities represents client capabilities
type ClientCapabilities struct {
	// Additional client capabilities can be added here
}

// Protocol implements the MCP protocol framework on top of a pluggable transport
// Including request/response linking, notifications, and progress features
type Protocol struct {
	transport            Transport
	requestMessageID     int
	requestHandlers      map[string]func(JSONRPCMessage, RequestHandlerExtra) (Result, error)
	notificationHandlers map[string]func(JSONRPCMessage) error
	responseHandlers     map[int]func(JSONRPCMessage, error)
	progressHandlers     map[int]ProgressCallback
	timeoutInfo          map[int]*TimeoutInfo
	options              *ProtocolOptions

	// For thread safety
	mutex sync.RWMutex

	// Callback functions
	onClose func()
	onError func(error)

	// Default handler for unregistered methods
	fallbackRequestHandler      func(Request) (Result, error)
	fallbackNotificationHandler func(Notification) error
}

// NewProtocol creates a new protocol instance
func NewProtocol(options *ProtocolOptions) *Protocol {
	p := &Protocol{
		requestMessageID:     1,
		requestHandlers:      make(map[string]func(JSONRPCMessage, RequestHandlerExtra) (Result, error)),
		notificationHandlers: make(map[string]func(JSONRPCMessage) error),
		responseHandlers:     make(map[int]func(JSONRPCMessage, error)),
		progressHandlers:     make(map[int]ProgressCallback),
		timeoutInfo:          make(map[int]*TimeoutInfo),
		options:              options,
	}

	// Set cancelled notification handler
	p.SetNotificationHandler("cancelled", p.handleCancelledNotification)

	// Set progress notification handler
	p.SetNotificationHandler("progress", p.handleProgressNotification)

	// Set automatic response ping handler
	p.SetRequestHandler("ping", func(msg JSONRPCMessage, extra RequestHandlerExtra) (Result, error) {
		return map[string]interface{}{}, nil
	})

	return p
}

// setupTimeout sets request timeout
func (p *Protocol) setupTimeout(messageID int, timeout int, maxTotalTimeout int, onTimeout func()) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	duration := time.Duration(timeout) * time.Millisecond

	timer := time.AfterFunc(duration, onTimeout)

	p.timeoutInfo[messageID] = &TimeoutInfo{
		timer:        timer,
		startTime:    time.Now(),
		timeout:      timeout,
		maxTotalTime: maxTotalTimeout,
		onTimeout:    onTimeout,
	}
}

// resetTimeout resets request timeout
func (p *Protocol) resetTimeout(messageID int) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	info, exists := p.timeoutInfo[messageID]
	if !exists {
		return false
	}

	// Check if it exceeds the maximum total time limit
	if info.maxTotalTime > 0 {
		elapsed := time.Since(info.startTime).Milliseconds()
		if elapsed >= int64(info.maxTotalTime) {
			info.timer.Stop()
			go info.onTimeout()
			return false
		}
	}

	// Reset timer
	info.timer.Reset(time.Duration(info.timeout) * time.Millisecond)
	return true
}

// cleanupTimeout cleans up timeout information
func (p *Protocol) cleanupTimeout(messageID int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if info, exists := p.timeoutInfo[messageID]; exists {
		info.timer.Stop()
		delete(p.timeoutInfo, messageID)
	}
}

// SetCloseHandler sets connection close callback
func (p *Protocol) SetCloseHandler(handler func()) {
	p.onClose = handler
}

// SetErrorHandler sets error callback
func (p *Protocol) SetErrorHandler(handler func(error)) {
	p.onError = handler
}

// SetTransport sets transport
func (p *Protocol) SetTransport(transport Transport) {
	p.transport = transport
	transport.SetCloseHandler(p.handleClose)
	transport.SetErrorHandler(p.handleError)
	transport.SetMessageHandler(p.handleMessage)
}

// Connect connects to the transport
func (p *Protocol) Connect(ctx context.Context, transport Transport) error {
	p.SetTransport(transport)
	return transport.Start(ctx)
}

// handleClose handles transport close event
func (p *Protocol) handleClose() {
	if p.onClose != nil {
		p.onClose()
	}
}

// handleError handles transport error event
func (p *Protocol) handleError(err error) {
	if p.onError != nil {
		p.onError(err)
	}
}

// handleMessage handles incoming messages
func (p *Protocol) handleMessage(message JSONRPCMessage) {
	// Determine message type (request, notification, or response)
	if _, ok := message["method"]; ok {
		// If there is a method field, this is either a request or notification
		if _, hasID := message["id"]; hasID {
			// With ID is a request
			p.handleRequest(message)
		} else {
			// Without ID is a notification
			p.handleNotification(message)
		}
	} else if _, hasID := message["id"]; hasID {
		// With ID but no method, this is a response
		p.handleResponse(message)
	} else {
		// Neither method nor ID, invalid message
		if p.onError != nil {
			p.onError(errors.New("received invalid message: missing both method and id"))
		}
	}
}

// handleNotification handles incoming notifications
func (p *Protocol) handleNotification(notification JSONRPCMessage) {
	method, _ := notification["method"].(string)

	p.mutex.RLock()
	handler, exists := p.notificationHandlers[method]
	p.mutex.RUnlock()

	if exists {
		go func() {
			if err := handler(notification); err != nil && p.onError != nil {
				p.onError(fmt.Errorf("notification handler error: %w", err))
			}
		}()
		return
	}

	// Use fallback handler
	if p.fallbackNotificationHandler != nil {
		var notif Notification
		notif.Method = method
		notif.Params = notification["params"]

		go func() {
			if err := p.fallbackNotificationHandler(notif); err != nil && p.onError != nil {
				p.onError(fmt.Errorf("fallback notification handler error: %w", err))
			}
		}()
	}
}

// handleCancelledNotification handles cancelled notification
func (p *Protocol) handleCancelledNotification(notification JSONRPCMessage) error {
	params, ok := notification["params"].(map[string]interface{})
	if !ok {
		return errors.New("invalid cancelled notification: params is not an object")
	}

	_, ok = params["requestId"]
	if !ok {
		return errors.New("invalid cancelled notification: missing requestId")
	}

	// Notify request handler that the request has been cancelled
	// In actual implementation, this would involve cancelling the context

	return nil
}

// handleProgressNotification handles progress notification
func (p *Protocol) handleProgressNotification(notification JSONRPCMessage) error {
	params, ok := notification["params"].(map[string]interface{})
	if !ok {
		return errors.New("invalid progress notification: params is not an object")
	}

	requestID, ok := params["requestId"]
	if !ok {
		return errors.New("invalid progress notification: missing requestId")
	}

	token, ok := requestID.(float64)
	if !ok {
		return errors.New("invalid progress notification: requestId is not a number")
	}

	messageID := int(token)

	// Reset timeout (if needed)
	p.mutex.RLock()
	info, exists := p.timeoutInfo[messageID]
	p.mutex.RUnlock()

	if exists && info != nil {
		p.resetTimeout(messageID)
	}

	// Get progress data
	progressData, ok := params["progress"].(map[string]interface{})
	if !ok {
		return errors.New("invalid progress notification: progress is not an object")
	}

	var progress Progress

	if msg, ok := progressData["message"].(string); ok {
		progress.Message = msg
	}

	if percent, ok := progressData["percent"].(float64); ok {
		progress.Percent = percent
	}

	if data, ok := progressData["data"]; ok {
		progress.Data = data
	}

	// Call progress callback
	p.mutex.RLock()
	callback, exists := p.progressHandlers[messageID]
	p.mutex.RUnlock()

	if exists && callback != nil {
		callback(progress)
	}

	return nil
}

// handleRequest handles incoming requests
func (p *Protocol) handleRequest(request JSONRPCMessage) {
	method, _ := request["method"].(string)

	p.mutex.RLock()
	handler, exists := p.requestHandlers[method]
	p.mutex.RUnlock()

	reqID := request["id"]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	extra := RequestHandlerExtra{
		Signal:    ctx,
		SessionID: p.transport.SessionID(),
	}

	go func() {
		var result Result
		var err error

		if exists {
			result, err = handler(request, extra)
		} else if p.fallbackRequestHandler != nil {
			var req Request
			req.Method = method
			req.Params = request["params"]
			result, err = p.fallbackRequestHandler(req)
		} else {
			err = &McpError{
				Code:    MethodNotFound,
				Message: fmt.Sprintf("Method not found: %s", method),
			}
		}

		// Build response
		response := JSONRPCMessage{
			"jsonrpc": "2.0",
			"id":      reqID,
		}

		if err != nil {
			var jsonRpcErr *JSONRPCError
			var mcpErr *McpError

			if errors.As(err, &jsonRpcErr) {
				response["error"] = jsonRpcErr
			} else if errors.As(err, &mcpErr) {
				response["error"] = JSONRPCError{
					Code:    int(mcpErr.Code),
					Message: mcpErr.Message,
					Data:    mcpErr.Data,
				}
			} else {
				response["error"] = JSONRPCError{
					Code:    int(InternalError),
					Message: err.Error(),
				}
			}
		} else {
			response["result"] = result
		}

		// Send response
		if sendErr := p.transport.Send(response); sendErr != nil && p.onError != nil {
			p.onError(fmt.Errorf("failed to send response: %w", sendErr))
		}
	}()
}

// handleResponse handles incoming responses
func (p *Protocol) handleResponse(response JSONRPCMessage) {
	idValue, ok := response["id"]
	if !ok {
		if p.onError != nil {
			p.onError(errors.New("received response without id"))
		}
		return
	}

	id, ok := idValue.(float64)
	if !ok {
		if p.onError != nil {
			p.onError(fmt.Errorf("received response with non-numeric id: %v", idValue))
		}
		return
	}

	messageID := int(id)

	// Clean up timeout
	p.cleanupTimeout(messageID)

	// Get response handler
	p.mutex.Lock()
	handler, exists := p.responseHandlers[messageID]
	if exists {
		delete(p.responseHandlers, messageID)
		delete(p.progressHandlers, messageID)
	}
	p.mutex.Unlock()

	if !exists {
		if p.onError != nil {
			p.onError(fmt.Errorf("received response for unknown request: %d", messageID))
		}
		return
	}

	// Check for error
	var err error
	if errorObj, hasError := response["error"]; hasError && errorObj != nil {
		errorMap, ok := errorObj.(map[string]interface{})
		if ok {
			code, _ := errorMap["code"].(float64)
			message, _ := errorMap["message"].(string)

			err = &JSONRPCError{
				Code:    int(code),
				Message: message,
				Data:    errorMap["data"],
			}
		} else {
			err = errors.New("invalid error object in response")
		}
	}

	// Call handler
	handler(response, err)
}

// Close closes the connection
func (p *Protocol) Close() error {
	if p.transport == nil {
		return errors.New("not connected")
	}

	return p.transport.Close()
}

// Transport returns the current used transport
func (p *Protocol) Transport() Transport {
	return p.transport
}

// Request sends a request and waits for a response
func (p *Protocol) Request(req Request, options *RequestOptions) (map[string]interface{}, error) {
	if p.transport == nil {
		return nil, errors.New("not connected")
	}

	// Generate message ID
	p.mutex.Lock()
	messageID := p.requestMessageID
	p.requestMessageID++
	p.mutex.Unlock()

	// Default timeout time
	timeout := DefaultRequestTimeoutMsec
	if options != nil && options.Timeout > 0 {
		timeout = options.Timeout
	}

	// Create json-rpc request message
	message := JSONRPCMessage{
		"jsonrpc": "2.0",
		"id":      messageID,
		"method":  req.Method,
	}

	if req.Params != nil {
		message["params"] = req.Params
	}

	// Set progress callback
	if options != nil && options.OnProgress != nil {
		p.mutex.Lock()
		p.progressHandlers[messageID] = options.OnProgress
		p.mutex.Unlock()
	}

	// Create result channel
	resultCh := make(chan struct {
		result map[string]interface{}
		err    error
	}, 1)

	// Set response handler
	p.mutex.Lock()
	p.responseHandlers[messageID] = func(response JSONRPCMessage, err error) {
		if err != nil {
			resultCh <- struct {
				result map[string]interface{}
				err    error
			}{nil, err}
			return
		}

		result, ok := response["result"]
		if !ok || result == nil {
			resultCh <- struct {
				result map[string]interface{}
				err    error
			}{nil, errors.New("response missing result")}
			return
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			resultCh <- struct {
				result map[string]interface{}
				err    error
			}{nil, fmt.Errorf("result is not an object: %v", result)}
			return
		}

		resultCh <- struct {
			result map[string]interface{}
			err    error
		}{resultMap, nil}
	}
	p.mutex.Unlock()

	// Set cancel handler
	var cancel context.CancelFunc = func() {}
	if options != nil && options.Signal != nil {
		// Listen for cancel when signal is present in options
		go func() {
			<-options.Signal.Done()
			// Send cancelled notification
			cancelNotification := JSONRPCMessage{
				"jsonrpc": "2.0",
				"method":  "cancelled",
				"params": map[string]interface{}{
					"requestId": messageID,
					"reason":    "cancelled by client",
				},
			}
			_ = p.transport.Send(cancelNotification)

			// Clean up
			p.mutex.Lock()
			delete(p.responseHandlers, messageID)
			delete(p.progressHandlers, messageID)
			p.mutex.Unlock()

			p.cleanupTimeout(messageID)

			resultCh <- struct {
				result map[string]interface{}
				err    error
			}{nil, context.Canceled}
		}()
	}

	// Set timeout handler
	maxTotalTimeout := 0
	if options != nil && options.MaxTotalTimeout > 0 {
		maxTotalTimeout = options.MaxTotalTimeout
	}

	p.setupTimeout(messageID, timeout, maxTotalTimeout, func() {
		p.mutex.Lock()
		delete(p.responseHandlers, messageID)
		delete(p.progressHandlers, messageID)
		p.mutex.Unlock()

		resultCh <- struct {
			result map[string]interface{}
			err    error
		}{nil, &McpError{
			Code:    RequestTimeout,
			Message: "Request timed out",
		}}
	})

	// Send request
	if err := p.transport.Send(message); err != nil {
		p.mutex.Lock()
		delete(p.responseHandlers, messageID)
		delete(p.progressHandlers, messageID)
		p.mutex.Unlock()

		p.cleanupTimeout(messageID)
		cancel()

		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for result or cancel
	result := <-resultCh

	if options != nil && options.Signal != nil && options.Signal.Err() == context.Canceled {
		return nil, context.Canceled
	}

	return result.result, result.err
}

// Notification sends a notification
func (p *Protocol) Notification(notification Notification) error {
	if p.transport == nil {
		return errors.New("not connected")
	}

	message := JSONRPCMessage{
		"jsonrpc": "2.0",
		"method":  notification.Method,
	}

	if notification.Params != nil {
		message["params"] = notification.Params
	}

	return p.transport.Send(message)
}

// SendProgressNotification sends a progress notification
func (p *Protocol) SendProgressNotification(requestID int, progress Progress) error {
	return p.Notification(Notification{
		Method: "progress",
		Params: map[string]interface{}{
			"requestId": requestID,
			"progress":  progress,
		},
	})
}

// SetRequestHandler sets request handler
func (p *Protocol) SetRequestHandler(method string, handler func(JSONRPCMessage, RequestHandlerExtra) (Result, error)) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.requestHandlers[method] = handler
}

// RemoveRequestHandler removes request handler
func (p *Protocol) RemoveRequestHandler(method string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.requestHandlers, method)
}

// SetNotificationHandler sets notification handler
func (p *Protocol) SetNotificationHandler(method string, handler func(JSONRPCMessage) error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.notificationHandlers[method] = handler
}

// RemoveNotificationHandler removes notification handler
func (p *Protocol) RemoveNotificationHandler(method string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.notificationHandlers, method)
}

// SetFallbackRequestHandler sets fallback request handler
func (p *Protocol) SetFallbackRequestHandler(handler func(Request) (Result, error)) {
	p.fallbackRequestHandler = handler
}

// SetFallbackNotificationHandler sets fallback notification handler
func (p *Protocol) SetFallbackNotificationHandler(handler func(Notification) error) {
	p.fallbackNotificationHandler = handler
}

// MergeCapabilities merges two capability objects
func MergeCapabilities(base, additional interface{}) (interface{}, error) {
	baseBytes, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal base capabilities: %w", err)
	}

	additionalBytes, err := json.Marshal(additional)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal additional capabilities: %w", err)
	}

	var baseMap map[string]interface{}
	if err := json.Unmarshal(baseBytes, &baseMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base capabilities: %w", err)
	}

	var additionalMap map[string]interface{}
	if err := json.Unmarshal(additionalBytes, &additionalMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal additional capabilities: %w", err)
	}

	// Merge maps
	result := make(map[string]interface{})

	for k, v := range baseMap {
		result[k] = v
	}

	for k, v := range additionalMap {
		if baseVal, exists := baseMap[k]; exists {
			// If both are maps, recursively merge
			baseValMap, baseIsMap := baseVal.(map[string]interface{})
			vMap, vIsMap := v.(map[string]interface{})

			if baseIsMap && vIsMap {
				merged := make(map[string]interface{})
				for baseKey, baseValue := range baseValMap {
					merged[baseKey] = baseValue
				}
				for vKey, vValue := range vMap {
					merged[vKey] = vValue
				}
				result[k] = merged
			} else {
				// Otherwise, use value from additional
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result, nil
}
