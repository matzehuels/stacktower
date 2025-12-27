package term

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Spinner provides a simple progress indicator.
type Spinner struct {
	message string
	done    chan struct{}
	stopped chan struct{}
	frames  []string
	mu      sync.Mutex
}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	go func() {
		defer close(s.stopped)
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				frame := s.frames[i%len(s.frames)]
				s.mu.Lock()
				fmt.Fprintf(os.Stderr, "\r%s %s", styleIconSpinner.Render(frame), StyleDim.Render(s.message))
				s.mu.Unlock()
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	close(s.done)
	<-s.stopped // Wait for goroutine to finish

	s.mu.Lock()
	// Clear the spinner line and add blank line for spacing
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(s.message)+4))
	s.mu.Unlock()
}

// StopWithSuccess stops the spinner and shows a success message.
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	PrintSuccess("%s", message)
}

// StopWithError stops the spinner and shows an error message.
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	PrintError("%s", message)
}
