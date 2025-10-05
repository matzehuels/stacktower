package httputil

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	retryErr := errors.New("retry me")

	tests := []struct {
		name         string
		maxAttempts  int
		fn           func(*int) func() error
		wantAttempts int
		wantErr      bool
	}{
		{
			name:        "successFirstAttempt",
			maxAttempts: 3,
			fn: func(attempts *int) func() error {
				return func() error {
					*attempts++
					return nil
				}
			},
			wantAttempts: 1,
		},
		{
			name:        "successAfterRetries",
			maxAttempts: 3,
			fn: func(attempts *int) func() error {
				return func() error {
					*attempts++
					if *attempts < 3 {
						return &RetryableError{Err: retryErr}
					}
					return nil
				}
			},
			wantAttempts: 3,
		},
		{
			name:        "maxRetriesExceeded",
			maxAttempts: 3,
			fn: func(attempts *int) func() error {
				return func() error {
					*attempts++
					return &RetryableError{Err: retryErr}
				}
			},
			wantAttempts: 3,
			wantErr:      true,
		},
		{
			name:        "nonRetryableError",
			maxAttempts: 3,
			fn: func(attempts *int) func() error {
				return func() error {
					*attempts++
					return errors.New("permanent")
				}
			},
			wantAttempts: 1,
			wantErr:      true,
		},
		{
			name:        "zeroAttemptsDefaultsToOne",
			maxAttempts: 0,
			fn: func(attempts *int) func() error {
				return func() error {
					*attempts++
					return nil
				}
			},
			wantAttempts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			err := Retry(context.Background(), tt.maxAttempts, time.Millisecond, tt.fn(&attempts))
			if (err != nil) != tt.wantErr {
				t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
			}
			if attempts != tt.wantAttempts {
				t.Errorf("got %d attempts, want %d", attempts, tt.wantAttempts)
			}
		})
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	err := Retry(ctx, 5, time.Millisecond, func() error {
		attempts++
		if attempts == 1 {
			cancel()
		}
		return &RetryableError{Err: errors.New("retry")}
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("got error %v, want context.Canceled", err)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	attempts := 0
	start := time.Now()

	err := Retry(context.Background(), 3, 50*time.Millisecond, func() error {
		attempts++
		if attempts < 3 {
			return &RetryableError{Err: errors.New("retry")}
		}
		return nil
	})

	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("got %d attempts, want 3", attempts)
	}
	minDelay := 150 * time.Millisecond // 50ms + 100ms
	if elapsed < minDelay {
		t.Errorf("elapsed %v too short, want >= %v", elapsed, minDelay)
	}
}

func TestRetryWithBackoff(t *testing.T) {
	attempts := 0
	err := RetryWithBackoff(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return &RetryableError{Err: errors.New("retry")}
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("got %d attempts, want 2", attempts)
	}
}

func TestRetryableError(t *testing.T) {
	inner := errors.New("inner")
	err := &RetryableError{Err: inner}

	if got := err.Error(); got != "inner" {
		t.Errorf("got Error() = %q, want %q", got, "inner")
	}
	if !errors.Is(err, inner) {
		t.Error("errors.Is failed to match wrapped error")
	}
	if got := errors.Unwrap(err); got != inner {
		t.Errorf("got Unwrap() = %v, want %v", got, inner)
	}
}
