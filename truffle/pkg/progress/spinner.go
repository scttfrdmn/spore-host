package progress

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type Spinner struct {
	frames  []string
	current int
	message string
	writer  io.Writer
	stopCh  chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex
}

func NewSpinner(w io.Writer, message string) *Spinner {
	return &Spinner{
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		message: message,
		writer:  w,
		stopCh:  make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.mu.Lock()
				_, _ = fmt.Fprintf(s.writer, "\r%s %s", s.frames[s.current], s.message)
				s.current = (s.current + 1) % len(s.frames)
				s.mu.Unlock()
			}
		}
	}()
}

func (s *Spinner) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	s.mu.Lock()
	_, _ = fmt.Fprint(s.writer, "\r\033[K") // Clear line
	s.mu.Unlock()
}

func (s *Spinner) UpdateMessage(msg string) {
	s.mu.Lock()
	s.message = msg
	s.mu.Unlock()
}
