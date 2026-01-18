package cli

import (
	"context"
	"io"

	"github.com/charmbracelet/log"
)

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

// ctxKey is the type for context keys used in this package.
type ctxKey int

const loggerKey ctxKey = 0

// WithLogger returns a new context with the given logger attached.
func WithLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// loggerFromContext retrieves the logger from ctx.
func loggerFromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerKey).(*Logger); ok {
		return l
	}
	return log.Default()
}
