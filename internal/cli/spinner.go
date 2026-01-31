package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Spinner provides a simple progress indicator with context cancellation support.
type Spinner struct {
	message string
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	stopped chan struct{}
	frames  []string
	mu      sync.Mutex
}

// newSpinner creates a new spinner with the given message.
func newSpinner(message string) *Spinner {
	return newSpinnerWithContext(context.Background(), message)
}

// newSpinnerWithContext creates a spinner that will stop when the context is cancelled.
func newSpinnerWithContext(ctx context.Context, message string) *Spinner {
	spinnerCtx, cancel := context.WithCancel(ctx)
	return &Spinner{
		message: message,
		ctx:     spinnerCtx,
		cancel:  cancel,
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	go func() {
		defer close(s.stopped)
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-s.ctx.Done():
				s.clearLine()
				return
			case <-s.done:
				return
			case <-ticker.C:
				frame := s.frames[i%len(s.frames)]
				s.mu.Lock()
				fmt.Fprintf(os.Stderr, "\r%s %s", styleIconSpinner.Render(frame), StyleDim.Render(s.message))
				s.mu.Unlock()
				i++
			}
		}
	}()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	s.cancel()
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	<-s.stopped
	s.clearLine()
}

func (s *Spinner) clearLine() {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(s.message)+4))
}

// StopWithSuccess stops the spinner and shows a success message.
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	printSuccess("%s", message)
}

// StopWithError stops the spinner and shows an error message.
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	printError("%s", message)
}

// Cancelled returns true if the spinner was stopped due to context cancellation.
func (s *Spinner) Cancelled() bool {
	return s.ctx.Err() != nil
}
