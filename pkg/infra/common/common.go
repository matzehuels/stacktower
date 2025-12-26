// Package common provides shared constants and utilities used across infra packages.
// This package has no dependencies on other infra packages to avoid import cycles.
package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

// =============================================================================
// Sentinel Errors
// =============================================================================

var (
	ErrNotFound  = errors.New("not found")
	ErrExpired   = errors.New("expired")
	ErrCacheMiss = errors.New("cache miss")
	ErrNetwork   = errors.New("network error")
	ErrInvalid   = errors.New("invalid")
)

// =============================================================================
// Cache TTLs
// =============================================================================

const (
	// GraphTTL is how long resolved dependency graphs are cached.
	GraphTTL = 7 * 24 * time.Hour // 7 days

	// LayoutTTL is how long computed layouts are cached.
	LayoutTTL = 30 * 24 * time.Hour // 30 days

	// RenderTTL is how long rendered artifacts (SVG/PNG/PDF) are cached.
	RenderTTL = 90 * 24 * time.Hour // 90 days

	// HTTPCacheTTL is the default TTL for HTTP response caching.
	HTTPCacheTTL = 24 * time.Hour

	// SessionTTL is the default session duration.
	SessionTTL = 24 * time.Hour

	// OAuthStateTTL is the default OAuth state token duration.
	OAuthStateTTL = 10 * time.Minute
)

// =============================================================================
// Hash Utilities
// =============================================================================

// HashBytes computes SHA256 hash of data and returns hex string.
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HashJSON computes SHA256 hash of JSON-serialized value.
func HashJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return HashBytes(data)
}

// HashShort returns first n characters of hash (for readable cache keys).
func HashShort(data []byte, n int) string {
	h := sha256.Sum256(data)
	s := hex.EncodeToString(h[:])
	if n > len(s) {
		n = len(s)
	}
	return s[:n]
}

// =============================================================================
// Logging
// =============================================================================

// LogLevel is a log level.
type LogLevel = log.Level

// Log levels.
const (
	LogDebug = log.DebugLevel
	LogInfo  = log.InfoLevel
	LogWarn  = log.WarnLevel
	LogError = log.ErrorLevel
)

// Logger is a structured logger.
type Logger = log.Logger

// NewLogger creates a new logger with timestamp formatting.
func NewLogger(w io.Writer, level LogLevel) *Logger {
	return log.NewWithOptions(w, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05.00",
		Level:           level,
	})
}

// NewStderrLogger creates a logger that writes to stderr at the given level.
func NewStderrLogger(level LogLevel) *Logger {
	return NewLogger(os.Stderr, level)
}

// DiscardLogger returns a logger that discards all output.
func DiscardLogger() *Logger {
	return log.NewWithOptions(io.Discard, log.Options{})
}

// DefaultLogger returns the default logger (writes to stderr at info level).
func DefaultLogger() *Logger {
	return log.Default()
}

// Progress tracks the start time of an operation and logs completion with elapsed duration.
type Progress struct {
	logger *Logger
	start  time.Time
}

// NewProgress creates a progress tracker that captures the current time as start.
func NewProgress(l *Logger) *Progress {
	return &Progress{logger: l, start: time.Now()}
}

// Done logs msg along with the elapsed time since Progress was created.
func (p *Progress) Done(msg string) {
	p.logger.Infof("%s (%s)", msg, time.Since(p.start).Round(time.Millisecond))
}

// Elapsed returns the time elapsed since Progress was created.
func (p *Progress) Elapsed() time.Duration {
	return time.Since(p.start)
}

// ctxKey is the type for context keys used in this package.
type ctxKey int

const loggerKey ctxKey = 0

// WithLogger returns a new context with the given logger attached.
func WithLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// LoggerFromContext retrieves the logger from ctx.
func LoggerFromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerKey).(*Logger); ok {
		return l
	}
	return DefaultLogger()
}

// =============================================================================
// Retry Utilities
// =============================================================================

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
// Only errors wrapped with [Retryable] will trigger retries.
func Retry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	attempts = max(attempts, 1)
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

// RetryWithBackoff is a convenience wrapper around [Retry] with sensible defaults.
// It retries up to 3 times with an initial 1-second delay and exponential backoff.
func RetryWithBackoff(ctx context.Context, fn func() error) error {
	return Retry(ctx, 3, time.Second, fn)
}

// IsRetryable checks if an error is wrapped with [RetryableError].
func IsRetryable(err error) bool {
	var re *RetryableError
	return errors.As(err, &re)
}
