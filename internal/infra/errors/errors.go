package errors

import "errors"

var (
    ErrUnauthorized = errors.New("unauthorized")
    ErrNotFound     = errors.New("resource not found")
    ErrForbidden    = errors.New("forbidden")
    ErrInvalidToken = errors.New("invalid token")
	ErrBlocked              = errors.New("user blocked")
	ErrAccountLocked        = errors.New("account locked due to too many failed login attempts")
)

var (
    ErrInvalidOperation = errors.New("Invalid Operation")
)