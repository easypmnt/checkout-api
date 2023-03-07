package websocketrpc

import "errors"

// Predefined errors.
var (
	ErrInvalidResponse  = errors.New("invalid response")
	ErrConnectionClosed = errors.New("connection closed")
)
