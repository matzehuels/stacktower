package cli

import (
	"context"
	"io"
	"time"

	"github.com/charmbracelet/log"
)

func newLogger(w io.Writer, level log.Level) *log.Logger {
	return log.NewWithOptions(w, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05.00",
		Level:           level,
	})
}

type progress struct {
	logger *log.Logger
	start  time.Time
}

func newProgress(l *log.Logger) *progress {
	return &progress{logger: l, start: time.Now()}
}

func (p *progress) done(msg string) {
	p.logger.Infof("%s (%s)", msg, time.Since(p.start).Round(time.Millisecond))
}

type ctxKey int

const loggerKey ctxKey = 0

func withLogger(ctx context.Context, l *log.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

func loggerFromContext(ctx context.Context) *log.Logger {
	if l, ok := ctx.Value(loggerKey).(*log.Logger); ok {
		return l
	}
	return log.Default()
}
