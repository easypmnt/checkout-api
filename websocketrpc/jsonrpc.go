package websocketrpc

import "encoding/json"

// Request represents a JSON-RPC notification
type Request struct {
	Version string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	Version string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Event represents a JSON-RPC event
type Event struct {
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}
