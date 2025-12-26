// Package logging provides shared logging utilities for CLI and API.
//
// This package wraps charmbracelet/log with common configuration and utilities
// used across the stacktower codebase. It provides:
//
//   - Consistent logger configuration with timestamps
//   - Progress tracking for long-running operations
//   - Context-based logger propagation
//
// # Usage
//
// Create a logger:
//
//	logger := logging.New(os.Stderr, logging.LevelInfo)
//	logger.Info("starting operation")
//
// Track progress:
//
//	prog := logging.NewProgress(logger)
//	// ... do work ...
//	prog.Done("processed 42 items")  // logs: "processed 42 items (1.234s)"
//
// Pass logger through context:
//
//	ctx = logging.WithLogger(ctx, logger)
//	// later...
//	logger := logging.FromContext(ctx)
package logging

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

// Level is a log level.
type Level = log.Level

// Log levels.
const (
	LevelDebug = log.DebugLevel
	LevelInfo  = log.InfoLevel
	LevelWarn  = log.WarnLevel
	LevelError = log.ErrorLevel
)

// Logger is a structured logger.
type Logger = log.Logger

// New creates a new logger with timestamp formatting.
// The logger writes to w and filters messages at the specified level.
// Timestamps are formatted as "HH:MM:SS.ms" (e.g., "14:32:01.45").
func New(w io.Writer, level Level) *Logger {
	return log.NewWithOptions(w, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05.00",
		Level:           level,
	})
}

// NewStderr creates a logger that writes to stderr at the given level.
func NewStderr(level Level) *Logger {
	return New(os.Stderr, level)
}

// Discard returns a logger that discards all output.
// Useful for testing or when logging should be disabled.
func Discard() *Logger {
	return log.NewWithOptions(io.Discard, log.Options{})
}

// Default returns the default logger (writes to stderr at info level).
func Default() *Logger {
	return log.Default()
}

// Progress tracks the start time of an operation and logs completion with elapsed duration.
// It is safe for sequential use by a single goroutine; concurrent calls to Done will race.
type Progress struct {
	logger *Logger
	start  time.Time
}

// NewProgress creates a progress tracker that captures the current time as start.
// The returned Progress should call Done when the operation completes.
func NewProgress(l *Logger) *Progress {
	return &Progress{logger: l, start: time.Now()}
}

// Done logs msg along with the elapsed time since Progress was created.
// The duration is rounded to the nearest millisecond.
// Example output: "Resolved 42 packages (1.234s)"
func (p *Progress) Done(msg string) {
	p.logger.Infof("%s (%s)", msg, time.Since(p.start).Round(time.Millisecond))
}

// Elapsed returns the time elapsed since Progress was created.
func (p *Progress) Elapsed() time.Duration {
	return time.Since(p.start)
}

// ctxKey is the type for context keys used in this package.
// Using a distinct type prevents collisions with other packages.
type ctxKey int

// loggerKey is the context key for storing a logger.
const loggerKey ctxKey = 0

// WithLogger returns a new context with the given logger attached.
// The logger can be retrieved later with FromContext.
func WithLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext retrieves the logger from ctx.
// If no logger is attached, it returns the default logger.
// This ensures callers always have a valid logger even if context setup fails.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerKey).(*Logger); ok {
		return l
	}
	return Default()
}
