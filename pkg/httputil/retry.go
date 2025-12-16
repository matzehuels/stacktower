package httputil

import (
	"context"
	"errors"
	"time"
)

type RetryableError struct{ Err error }

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

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

func RetryWithBackoff(ctx context.Context, fn func() error) error {
	return Retry(ctx, 3, time.Second, fn)
}

func isRetryable(err error) bool {
	return errors.As(err, new(*RetryableError))
}
