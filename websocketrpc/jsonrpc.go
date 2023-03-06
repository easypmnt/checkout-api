package websocketrpc

import (
	"encoding/json"
	"fmt"
)

// Request represents a JSON-RPC notification
type Request struct {
	Version string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	Version string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *Error) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

// Event represents a JSON-RPC event
type Event struct {
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// messagePayload represents a JSON-RPC response/event payload
type messagePayload struct {
	Version string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IsEvent returns true if the message is an event
func (m *messagePayload) IsEvent() bool {
	return m.Method != "" && m.ID == nil
}

// IsResponse returns true if the message is a response
func (m *messagePayload) IsResponse() bool {
	return m.Method == "" && m.ID != nil
}
