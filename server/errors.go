package server

import (
	"errors"
	"net/http"

	"github.com/easypmnt/checkout-api/internal/httpencoder"
)

// Predefined errors.
var (
	ErrInvalidRequest   = errors.New("invalid_request")
	ErrInvalidParameter = errors.New("invalid_parameter")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not_found")
)

// Error codes map
var ErrorCodes = map[error]int{
	ErrInvalidRequest:   http.StatusBadRequest,
	ErrInvalidParameter: http.StatusBadRequest,
	ErrForbidden:        http.StatusForbidden,
	ErrNotFound:         http.StatusNotFound,
}

// Error messages
var ErrorMessages = map[error]string{
	ErrInvalidRequest:   "Invalid request payload",
	ErrInvalidParameter: "Some parameters are invalid",
	ErrForbidden:        "Forbidden. You don't have permission to access this account",
	ErrNotFound:         "Not found",
}

// NewError creates a new error
func NewError(err error) *httpencoder.ErrorResponse {
	code, ok := ErrorCodes[err]
	if !ok {
		if stdErr := findError(err); stdErr != nil {
			code, ok = ErrorCodes[stdErr]
			if !ok {
				code = http.StatusInternalServerError
			}
		} else {
			return nil
		}
	}

	errStr := err.Error()
	msg, ok := ErrorMessages[err]
	if !ok {
		errStr = http.StatusText(code)
		msg = err.Error()
	}

	return &httpencoder.ErrorResponse{
		Code:    code,
		Error:   errStr,
		Message: msg,
	}
}

func findError(err error) error {
	for stdErr := range ErrorCodes {
		if errors.Is(err, stdErr) {
			return stdErr
		}
	}
	return nil
}
