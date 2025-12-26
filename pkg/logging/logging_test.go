package logging

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, LevelInfo)

	if logger == nil {
		t.Fatal("New() returned nil")
	}

	logger.Info("test message")

	if buf.Len() == 0 {
		t.Error("logger should have written output")
	}
}

func TestNewLevels(t *testing.T) {
	tests := []struct {
		name    string
		level   Level
		logFunc func(*Logger)
		wantLog bool
	}{
		{
			name:    "info at info level",
			level:   LevelInfo,
			logFunc: func(l *Logger) { l.Info("test") },
			wantLog: true,
		},
		{
			name:    "debug at info level",
			level:   LevelInfo,
			logFunc: func(l *Logger) { l.Debug("test") },
			wantLog: false,
		},
		{
			name:    "debug at debug level",
			level:   LevelDebug,
			logFunc: func(l *Logger) { l.Debug("test") },
			wantLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&buf, tt.level)
			tt.logFunc(logger)

			gotLog := buf.Len() > 0
			if gotLog != tt.wantLog {
				t.Errorf("got log output = %v, want %v", gotLog, tt.wantLog)
			}
		})
	}
}

func TestDiscard(t *testing.T) {
	logger := Discard()
	if logger == nil {
		t.Fatal("Discard() returned nil")
	}

	// Should not panic
	logger.Info("test")
	logger.Debug("test")
	logger.Error("test")
}

func TestProgress(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, LevelInfo)

	prog := NewProgress(logger)
	if prog == nil {
		t.Fatal("NewProgress() returned nil")
	}

	// Small delay to ensure measurable duration
	time.Sleep(10 * time.Millisecond)

	prog.Done("test completed")

	output := buf.String()
	if output == "" {
		t.Error("Progress.Done() should produce output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("test completed")) {
		t.Error("Progress.Done() output should contain message")
	}
}

func TestProgressElapsed(t *testing.T) {
	logger := Discard()
	prog := NewProgress(logger)

	time.Sleep(10 * time.Millisecond)

	elapsed := prog.Elapsed()
	if elapsed < 10*time.Millisecond {
		t.Errorf("Elapsed() = %v, want >= 10ms", elapsed)
	}
}

func TestWithLogger(t *testing.T) {
	ctx := context.Background()
	logger := Default()

	ctxWithLogger := WithLogger(ctx, logger)

	retrieved := FromContext(ctxWithLogger)
	if retrieved != logger {
		t.Error("FromContext should return the same logger")
	}
}

func TestFromContextDefault(t *testing.T) {
	ctx := context.Background()

	logger := FromContext(ctx)
	if logger == nil {
		t.Error("FromContext should return default logger when none set")
	}
}

func TestFromContextWithValue(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	customLogger := New(&buf, LevelInfo)

	ctx = WithLogger(ctx, customLogger)
	retrieved := FromContext(ctx)

	if retrieved != customLogger {
		t.Error("FromContext should return the custom logger")
	}

	retrieved.Info("test")
	if buf.Len() == 0 {
		t.Error("custom logger should write to buffer")
	}
}
