package logsource

import (
	"bufio"
	"context"
	"os"
)

// StdinSource reads log lines from stdin.
type StdinSource struct {
	ch     chan string
	cancel context.CancelFunc
}

// NewStdinSource creates a StdinSource that reads from stdin in a background goroutine.
func NewStdinSource(ctx context.Context) *StdinSource {
	ctx, cancel := context.WithCancel(ctx)
	s := &StdinSource{
		ch:     make(chan string, 50000),
		cancel: cancel,
	}
	go s.read(ctx)
	return s
}

func (s *StdinSource) read(ctx context.Context) {
	defer close(s.ch)

	scanner := bufio.NewScanner(os.Stdin)

	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	scanChan := make(chan bool, 1)

	for {
		go func() {
			scanChan <- scanner.Scan()
		}()

		select {
		case <-ctx.Done():
			return
		case hasLine := <-scanChan:
			if !hasLine {
				return
			}
			line := scanner.Text()
			if line != "" {
				select {
				case s.ch <- line:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (s *StdinSource) Lines() <-chan string { return s.ch }
func (s *StdinSource) Stop()               { s.cancel() }
func (s *StdinSource) Name() string        { return "stdin" }
