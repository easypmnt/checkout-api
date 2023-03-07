package websocketrpc

import (
	"encoding/json"
	"fmt"
)

// Request represents a JSON-RPC notification
type Request struct {
	Version string      `json:"jsonrpc"`
	ID      uint64      `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	Version string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
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
	if e == nil || e.Code == 0 {
		return ""
	}
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

// Event represents a JSON-RPC event
type Event struct {
	Version string       `json:"jsonrpc"`
	Method  string       `json:"method"`
	Params  *EventParams `json:"params,omitempty"`
}

// EventParams represents the params of a JSON-RPC event
type EventParams struct {
	Result       json.RawMessage `json:"result,omitempty"`
	Subscription json.Number     `json:"subscription,omitempty"`
}

// messagePayload represents a JSON-RPC response/event payload
type messagePayload struct {
	Version string          `json:"jsonrpc"`
	ID      json.Number     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  *EventParams    `json:"params,omitempty"`
}

// IsEvent returns true if the message is an event
func (m *messagePayload) IsEvent() bool {
	id, _ := m.ID.Int64()
	return id == 0 && m.Method != ""
}

// IsResponse returns true if the message is a response
func (m *messagePayload) IsResponse() bool {
	id, err := m.ID.Int64()
	return err == nil && id > 0 && m.Method == ""
}

// GetResponse returns a Response object from a messagePayload
func (m *messagePayload) GetResponse() *Response {
	id, _ := m.ID.Int64()
	return &Response{
		Version: m.Version,
		ID:      uint64(id),
		Result:  m.Result,
		Error:   m.Error,
	}
}

// GetEvent returns an Event object from a messagePayload
func (m *messagePayload) GetEvent() *Event {
	return &Event{
		Version: m.Version,
		Method:  m.Method,
		Params:  m.Params,
	}
}
