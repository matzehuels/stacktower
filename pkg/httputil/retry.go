package httputil

import (
	"context"
	"errors"
	"time"
)

// RetryableError wraps an error to indicate it should trigger a retry.
// Use this type to signal transient failures like network timeouts,
// temporary DNS resolution failures, or HTTP 5xx server errors.
// Errors not wrapped in RetryableError are treated as permanent failures
// and cause [Retry] to return immediately without further attempts.
//
// Prefer using the [Retryable] helper function for convenience:
//
//	if resp.StatusCode >= 500 {
//	    return httputil.Retryable(fmt.Errorf("server error: %d", resp.StatusCode))
//	}
//
// RetryableError implements error unwrapping, so errors.Is and errors.As
// work correctly with the wrapped error.
type RetryableError struct{ Err error }

// Retryable wraps an error as a [RetryableError], signaling to [Retry]
// that this failure should trigger a retry attempt.
//
// This is a convenience helper that avoids verbose struct literal syntax.
// Returns nil if err is nil, allowing safe use in error returns:
//
//	if err := doSomething(); err != nil {
//	    return httputil.Retryable(err)
//	}
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// Error returns the error message of the wrapped error.
func (e *RetryableError) Error() string { return e.Err.Error() }

// Unwrap returns the wrapped error, enabling errors.Is and errors.As
// to inspect the underlying cause.
func (e *RetryableError) Unwrap() error { return e.Err }

// Retry executes fn up to attempts times with exponential backoff.
//
// Only errors wrapped with [RetryableError] trigger a retry; all other errors
// are returned immediately. Between retries, Retry waits for delay, then
// doubles the delay for the next attempt (1s, 2s, 4s, etc.). If ctx is
// cancelled during a retry delay, Retry returns ctx.Err() immediately.
//
// Parameters:
//   - ctx: Context for cancellation. If cancelled during backoff, returns ctx.Err().
//   - attempts: Maximum number of attempts (minimum 1). Zero or negative values default to 1.
//   - delay: Initial backoff duration. Doubled after each failed attempt.
//   - fn: Function to execute. Wrap errors in [RetryableError] to enable retries.
//
// Returns the result of fn on success, the last error if all attempts fail,
// or ctx.Err() if the context is cancelled during backoff.
//
// Retry is safe to call from multiple goroutines. However, fn itself must
// handle any concurrency concerns for the operation it performs.
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

// RetryWithBackoff is a convenience wrapper around [Retry] with sensible defaults.
//
// It performs up to 3 attempts with exponential backoff starting at 1 second:
// attempt 1 (immediate), wait 1s, attempt 2, wait 2s, attempt 3.
// Total maximum wait time is 3 seconds across all retries.
//
// Use this when you need retry logic but don't need custom retry parameters.
// For more control over attempts or delay, call [Retry] directly.
func RetryWithBackoff(ctx context.Context, fn func() error) error {
	return Retry(ctx, 3, time.Second, fn)
}

func isRetryable(err error) bool {
	return errors.As(err, new(*RetryableError))
}
