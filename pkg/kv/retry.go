package kv

import (
	"context"
	"errors"
	"time"
)

// RetryableError wraps an error to indicate it should trigger a retry.
type RetryableError struct{ Err error }

// Retryable wraps an error as a [RetryableError].
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

// Retry executes fn up to attempts times with exponential backoff.
func Retry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	attempts = max(attempts, 1)
	var lastErr error

	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else if lastErr = err; !isRetryable(err) {
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

// RetryWithBackoff is a convenience wrapper around [Retry] with sensible defaults.
func RetryWithBackoff(ctx context.Context, fn func() error) error {
	return Retry(ctx, 3, time.Second, fn)
}

func isRetryable(err error) bool {
	var re *RetryableError
	return errors.As(err, &re)
}
