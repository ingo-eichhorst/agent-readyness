package pipeline

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
)

// ProgressFunc is a callback for pipeline stage progress updates.
type ProgressFunc func(stage string, detail string)

// Spinner displays an animated spinner on stderr for long-running operations.
// It is automatically suppressed when stderr is not a TTY (piped output, CI).
type Spinner struct {
	mu      sync.Mutex
	frames  []string
	current int
	message string
	active  bool
	isTTY   bool
	writer  *os.File
	ticker  *time.Ticker
	done    chan struct{}
}

// NewSpinner creates a new Spinner that writes to the given file (typically os.Stderr).
func NewSpinner(w *os.File) *Spinner {
	return &Spinner{
		frames: []string{"|", "/", "-", "\\"},
		writer: w,
		isTTY:  isatty.IsTerminal(w.Fd()) || isatty.IsCygwinTerminal(w.Fd()),
		done:   make(chan struct{}),
	}
}

// Start begins displaying the spinner with the given message.
// If the writer is not a TTY, Start is a no-op.
func (s *Spinner) Start(message string) {
	if !s.isTTY {
		return
	}

	s.mu.Lock()
	s.active = true
	s.message = message
	s.mu.Unlock()

	const spinnerInterval = 100 * time.Millisecond
	s.ticker = time.NewTicker(spinnerInterval)
	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.mu.Lock()
				if !s.active {
					s.mu.Unlock()
					return
				}
				frame := s.frames[s.current%len(s.frames)]
				msg := s.message
				s.current++
				s.mu.Unlock()
				fmt.Fprintf(s.writer, "\r%s %s", frame, msg)
			}
		}
	}()
}

// Update changes the spinner message. The next tick will display the new message.
func (s *Spinner) Update(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// Stop halts the spinner and optionally prints a final message.
// If the writer is not a TTY, Stop is a no-op.
func (s *Spinner) Stop(finalMessage string) {
	if !s.isTTY {
		return
	}

	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.done)

	if finalMessage != "" {
		fmt.Fprintf(s.writer, "\r%s\n", finalMessage)
	} else {
		// Clear the spinner line
		fmt.Fprintf(s.writer, "\r\033[K")
	}
}
