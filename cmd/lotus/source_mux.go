package main

import (
	"context"
	"sync"

	"github.com/tinytelemetry/lotus/internal/model"
)

// DefaultMuxBuffer is the default channel buffer size for the source multiplexer.
const DefaultMuxBuffer = 50_000

// SourceMultiplexer merges multiple log sources into a single read-only stream.
type SourceMultiplexer struct {
	ctx    context.Context
	cancel context.CancelFunc

	sources []NamedLogSource
	lines   chan model.IngestEnvelope

	startOnce sync.Once
	stopOnce  sync.Once
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func NewSourceMultiplexer(parent context.Context, sources []NamedLogSource, buffer int) *SourceMultiplexer {
	if buffer <= 0 {
		buffer = DefaultMuxBuffer
	}
	ctx, cancel := context.WithCancel(parent)
	return &SourceMultiplexer{
		ctx:     ctx,
		cancel:  cancel,
		sources: sources,
		lines:   make(chan model.IngestEnvelope, buffer),
	}
}

func (m *SourceMultiplexer) Start() {
	m.startOnce.Do(func() {
		if len(m.sources) == 0 {
			m.closeOutput()
			return
		}

		for _, src := range m.sources {
			src := src
			m.wg.Add(1)
			go m.forward(src)
		}

		go func() {
			m.wg.Wait()
			m.closeOutput()
		}()
	})
}

func (m *SourceMultiplexer) Stop() {
	m.stopOnce.Do(func() {
		m.cancel()
		for _, src := range m.sources {
			src.Stop()
		}
		m.wg.Wait()
		m.closeOutput()
	})
}

func (m *SourceMultiplexer) HasSources() bool {
	return len(m.sources) > 0
}

func (m *SourceMultiplexer) PrimarySourceName() string {
	if len(m.sources) == 0 {
		return ""
	}
	return m.sources[0].Name()
}

func (m *SourceMultiplexer) Lines() <-chan model.IngestEnvelope {
	return m.lines
}

func (m *SourceMultiplexer) forward(src NamedLogSource) {
	defer m.wg.Done()

	sourceLines := src.Lines()
	for {
		select {
		case <-m.ctx.Done():
			return
		case line, ok := <-sourceLines:
			if !ok {
				return
			}
			if line.Line == "" {
				continue
			}
			select {
			case m.lines <- line:
			case <-m.ctx.Done():
				return
			}
		}
	}
}

func (m *SourceMultiplexer) closeOutput() {
	m.closeOnce.Do(func() {
		close(m.lines)
	})
}
