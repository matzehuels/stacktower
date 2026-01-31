package cache

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors for caching operations.
var (
	// ErrNotFound is returned when a requested item does not exist.
	ErrNotFound = errors.New("not found")

	// ErrNetwork is returned for HTTP failures (timeouts, connection errors, 5xx responses).
	ErrNetwork = errors.New("network error")

	// ErrCacheMiss is returned when an item is not found in cache.
	ErrCacheMiss = errors.New("cache miss")
)

// RetryableError wraps an error to indicate it should trigger a retry.
type RetryableError struct{ Err error }

// Retryable wraps an error as a RetryableError.
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// Error returns the error message of the wrapped error.
func (e *RetryableError) Error() string { return e.Err.Error() }

// Unwrap returns the wrapped error.
func (e *RetryableError) Unwrap() error { return e.Err }

// IsRetryable checks if an error is wrapped with RetryableError.
func IsRetryable(err error) bool {
	var re *RetryableError
	return errors.As(err, &re)
}

// RetryWithBackoff retries fn up to 3 times with exponential backoff.
// Only errors wrapped with Retryable will trigger retries.
func RetryWithBackoff(ctx context.Context, fn func() error) error {
	const attempts = 3
	delay := time.Second
	var lastErr error

	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else if lastErr = err; !IsRetryable(err) {
			return err
		}

		if i < attempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay *= 2
			}
		}
	}
	return lastErr
}
