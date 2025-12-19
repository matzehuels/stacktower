package cli

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf, log.InfoLevel)

	if logger == nil {
		t.Fatal("newLogger() returned nil")
	}

	// Test that it can log
	logger.Info("test message")

	if buf.Len() == 0 {
		t.Error("logger should have written output")
	}
}

func TestNewLoggerLevels(t *testing.T) {
	tests := []struct {
		name    string
		level   log.Level
		logFunc func(*log.Logger)
		wantLog bool
	}{
		{
			name:    "info at info level",
			level:   log.InfoLevel,
			logFunc: func(l *log.Logger) { l.Info("test") },
			wantLog: true,
		},
		{
			name:    "debug at info level",
			level:   log.InfoLevel,
			logFunc: func(l *log.Logger) { l.Debug("test") },
			wantLog: false,
		},
		{
			name:    "debug at debug level",
			level:   log.DebugLevel,
			logFunc: func(l *log.Logger) { l.Debug("test") },
			wantLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := newLogger(&buf, tt.level)
			tt.logFunc(logger)

			gotLog := buf.Len() > 0
			if gotLog != tt.wantLog {
				t.Errorf("got log output = %v, want %v", gotLog, tt.wantLog)
			}
		})
	}
}

func TestProgress(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf, log.InfoLevel)

	prog := newProgress(logger)
	if prog == nil {
		t.Fatal("newProgress() returned nil")
	}

	// Small delay to ensure measurable duration
	time.Sleep(10 * time.Millisecond)

	prog.done("test completed")

	output := buf.String()
	if output == "" {
		t.Error("progress.done() should produce output")
	}

	// Should contain the message
	if !bytes.Contains(buf.Bytes(), []byte("test completed")) {
		t.Error("progress.done() output should contain message")
	}
}

func TestWithLogger(t *testing.T) {
	ctx := context.Background()
	logger := log.Default()

	ctxWithLogger := withLogger(ctx, logger)

	// Should be able to retrieve the logger
	retrieved := loggerFromContext(ctxWithLogger)
	if retrieved != logger {
		t.Error("loggerFromContext should return the same logger")
	}
}

func TestLoggerFromContextDefault(t *testing.T) {
	ctx := context.Background()

	// Without logger in context, should return default
	logger := loggerFromContext(ctx)
	if logger == nil {
		t.Error("loggerFromContext should return default logger when none set")
	}
}

func TestLoggerFromContextWithValue(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	customLogger := newLogger(&buf, log.InfoLevel)

	ctx = withLogger(ctx, customLogger)
	retrieved := loggerFromContext(ctx)

	if retrieved != customLogger {
		t.Error("loggerFromContext should return the custom logger")
	}

	// Verify it works by logging
	retrieved.Info("test")
	if buf.Len() == 0 {
		t.Error("custom logger should write to buffer")
	}
}
