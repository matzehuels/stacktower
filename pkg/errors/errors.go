package errors

import "errors"

// Sentinel errors used across packages
var (
	ErrNotFound  = errors.New("not found")
	ErrExpired   = errors.New("expired")
	ErrCacheMiss = errors.New("cache miss")
	ErrNetwork   = errors.New("network error")
	ErrInvalid   = errors.New("invalid")
)
