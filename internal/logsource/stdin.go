package logsource

import (
	"bufio"
	"context"
	"errors"
	"log"
	"os"

	"github.com/tinytelemetry/lotus/internal/model"
)

const (
	// DefaultStdinBuffer is the default channel buffer size for stdin lines.
	DefaultStdinBuffer = 50_000

	// DefaultStdinMaxLineSize is the default maximum size (in bytes) of a single stdin line.
	DefaultStdinMaxLineSize = 1024 * 1024 // 1MB
)

// StdinConfig holds tunable parameters for the stdin source.
type StdinConfig struct {
	BufferSize  int
	MaxLineSize int
}

// StdinSource reads log lines from stdin.
type StdinSource struct {
	ch     chan model.IngestEnvelope
	cancel context.CancelFunc
}

// NewStdinSource creates a StdinSource that reads from stdin in a background goroutine.
func NewStdinSource(ctx context.Context, conf ...StdinConfig) *StdinSource {
	bufferSize := DefaultStdinBuffer
	maxLineSize := DefaultStdinMaxLineSize
	if len(conf) > 0 {
		if conf[0].BufferSize > 0 {
			bufferSize = conf[0].BufferSize
		}
		if conf[0].MaxLineSize > 0 {
			maxLineSize = conf[0].MaxLineSize
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	s := &StdinSource{
		ch:     make(chan model.IngestEnvelope, bufferSize),
		cancel: cancel,
	}
	go s.read(ctx, maxLineSize)
	return s
}

func (s *StdinSource) read(ctx context.Context, maxLineSize int) {
	defer close(s.ch)

	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, maxLineSize)
	scanner.Buffer(buf, maxLineSize)

	// Use a single goroutine for blocking scan with a done channel to
	// detect context cancellation without spawning a goroutine per line.
	type scanResult struct {
		line string
		ok   bool
	}
	results := make(chan scanResult)
	go func() {
		defer close(results)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			select {
			case results <- scanResult{line: line, ok: true}:
			case <-ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil {
			if errors.Is(err, bufio.ErrTooLong) {
				log.Printf("logsource: stdin line exceeded max size (%d bytes), stopping stdin source", maxLineSize)
				return
			}
			log.Printf("logsource: stdin scanner error: %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-results:
			if !ok || !r.ok {
				return
			}
			select {
			case s.ch <- model.IngestEnvelope{Source: s.Name(), Line: r.line}:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (s *StdinSource) Lines() <-chan model.IngestEnvelope { return s.ch }
func (s *StdinSource) Stop()                              { s.cancel() }
func (s *StdinSource) Name() string                       { return "stdin" }
