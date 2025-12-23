// Package cli implements the stacktower command-line interface.
//
// This package provides commands for parsing dependency graphs from various
// package managers, rendering them as visualizations, and managing the HTTP
// response cache. The CLI is built using cobra and supports verbose logging
// via the charmbracelet/log library.
//
// # Commands
//
// The main commands are:
//   - parse: Extract dependency graphs from package managers or manifest files
//   - render: Generate SVG, PDF, or PNG visualizations
//   - cache: Manage the HTTP response cache
//   - pqtree: Debug tool for PQ-tree constraint visualization
//
// # Logging
//
// All commands support --verbose (-v) for debug-level logging. Loggers are
// passed through context.Context to allow structured progress tracking.
//
// # Example
//
//	import "github.com/matzehuels/stacktower/internal/cli"
//
//	func main() {
//	    if err := cli.Execute(); err != nil {
//	        os.Exit(1)
//	    }
//	}
package cli

import (
	"context"
	"io"
	"time"

	"github.com/charmbracelet/log"
)

// newLogger creates a new logger with timestamp formatting.
// The logger writes to w and filters messages at the specified level.
// Timestamps are formatted as "HH:MM:SS.ms" (e.g., "14:32:01.45").
func newLogger(w io.Writer, level log.Level) *log.Logger {
	return log.NewWithOptions(w, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05.00",
		Level:           level,
	})
}

// progress tracks the start time of an operation and logs completion with elapsed duration.
// It is safe for sequential use by a single goroutine; concurrent calls to done will race.
type progress struct {
	logger *log.Logger
	start  time.Time
}

// newProgress creates a progress tracker that captures the current time as start.
// The returned progress should call done when the operation completes.
func newProgress(l *log.Logger) *progress {
	return &progress{logger: l, start: time.Now()}
}

// done logs msg along with the elapsed time since progress was created.
// The duration is rounded to the nearest millisecond.
// Example output: "Resolved 42 packages (1.234s)"
func (p *progress) done(msg string) {
	p.logger.Infof("%s (%s)", msg, time.Since(p.start).Round(time.Millisecond))
}

// ctxKey is the type for context keys used in this package.
// Using a distinct type prevents collisions with other packages.
type ctxKey int

// loggerKey is the context key for storing a logger.
const loggerKey ctxKey = 0

// withLogger returns a new context with the given logger attached.
// The logger can be retrieved later with loggerFromContext.
func withLogger(ctx context.Context, l *log.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// loggerFromContext retrieves the logger from ctx.
// If no logger is attached, it returns log.Default().
// This ensures commands always have a valid logger even if context setup fails.
func loggerFromContext(ctx context.Context) *log.Logger {
	if l, ok := ctx.Value(loggerKey).(*log.Logger); ok {
		return l
	}
	return log.Default()
}
