package logsource

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sync"

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
	ch       chan model.IngestEnvelope
	cancel   context.CancelFunc
	reader   io.ReadCloser
	wg       sync.WaitGroup
	stopOnce sync.Once
}

// NewStdinSource creates a StdinSource that reads from stdin in a background goroutine.
func NewStdinSource(ctx context.Context, conf ...StdinConfig) *StdinSource {
	return newStdinSourceWithReader(ctx, os.Stdin, conf...)
}

func newStdinSourceWithReader(ctx context.Context, reader io.ReadCloser, conf ...StdinConfig) *StdinSource {
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
		reader: reader,
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.read(ctx, reader, maxLineSize)
	}()
	return s
}

func (s *StdinSource) read(ctx context.Context, reader io.Reader, maxLineSize int) {
	defer close(s.ch)

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, maxLineSize)
	scanner.Buffer(buf, maxLineSize)

	for {
		if !scanner.Scan() {
			if ctx.Err() != nil {
				return
			}
			if err := scanner.Err(); err != nil {
				if errors.Is(err, bufio.ErrTooLong) {
					log.Printf("logsource: stdin line exceeded max size (%d bytes), stopping stdin source", maxLineSize)
					return
				}
				log.Printf("logsource: stdin scanner error: %v", err)
			}
			return
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		select {
		case s.ch <- model.IngestEnvelope{Source: s.Name(), Line: line}:
		case <-ctx.Done():
			return
		}
	}
}

func (s *StdinSource) Lines() <-chan model.IngestEnvelope { return s.ch }
func (s *StdinSource) Stop() {
	s.stopOnce.Do(func() {
		s.cancel()
		if s.reader != nil {
			_ = s.reader.Close()
		}
		s.wg.Wait()
	})
}
func (s *StdinSource) Name() string { return "stdin" }
