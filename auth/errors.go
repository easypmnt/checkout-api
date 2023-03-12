package auth

import "errors"

// Predefined errors.
var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrInvalidToken         = errors.New("invalid token")
	ErrInvalidGrantType     = errors.New("invalid grant type")
	ErrPasswordNotSupported = errors.New("password grant type not supported")
	ErrTokenExpired         = errors.New("token expired")
)
