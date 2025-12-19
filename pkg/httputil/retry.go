package httputil

import (
	"context"
	"errors"
	"time"
)

// RetryableError wraps an error to indicate it should trigger a retry.
// Wrap transient failures (network timeouts, 5xx responses) with this type
// so that [Retry] knows to attempt the operation again.
type RetryableError struct{ Err error }

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

// Retry executes fn up to attempts times with exponential backoff.
// It only retries errors wrapped with [RetryableError]; other errors are
// returned immediately. The delay doubles after each failed attempt.
// Returns the last error if all attempts fail, or ctx.Err() if cancelled.
func Retry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	attempts = max(attempts, 1)
	var lastErr error

	for i := range attempts {
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

// RetryWithBackoff is a convenience wrapper around [Retry] with sensible
// defaults: 3 attempts with 1 second initial delay (doubling each retry).
func RetryWithBackoff(ctx context.Context, fn func() error) error {
	return Retry(ctx, 3, time.Second, fn)
}

func isRetryable(err error) bool {
	return errors.As(err, new(*RetryableError))
}
