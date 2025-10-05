package httputil

import (
	"context"
	"errors"
	"time"
)

type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

func Retry(ctx context.Context, max int, delay time.Duration, fn func() error) error {
	if max < 1 {
		max = 1
	}

	var lastErr error
	for attempt := 0; attempt < max; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else if lastErr = err; !errors.As(err, new(*RetryableError)) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= 2
		}
	}
	return lastErr
}

func RetryWithBackoff(ctx context.Context, fn func() error) error {
	return Retry(ctx, 3, time.Second, fn)
}
