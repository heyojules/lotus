package ingest

import (
	"sync"

	"github.com/control-theory/lotus/internal/model"
)

// PassthroughProcessor is a lightweight processor that avoids JSON parsing.
// It creates fallback records directly from input lines.
type PassthroughProcessor struct {
	mu         sync.RWMutex
	sink       RecordSink
	sourceName string
}

// NewPassthroughProcessor creates a new passthrough processor.
func NewPassthroughProcessor(sink RecordSink, sourceName string) *PassthroughProcessor {
	return &PassthroughProcessor{
		sink:       sink,
		sourceName: sourceName,
	}
}

func (p *PassthroughProcessor) Name() string { return ProcessorModePassthrough }

// ProcessLine processes an untagged line using the processor source name.
func (p *PassthroughProcessor) ProcessLine(line string) *ProcessResult {
	return p.ProcessEnvelope(model.IngestEnvelope{
		Source: p.getSourceName(),
		Line:   line,
	})
}

// ProcessEnvelope processes one source-tagged line.
func (p *PassthroughProcessor) ProcessEnvelope(env model.IngestEnvelope) *ProcessResult {
	if env.Line == "" {
		return nil
	}

	source := env.Source
	if source == "" {
		source = p.getSourceName()
	}

	record := CreateFallbackLogEntry(env.Line)
	record.Source = source

	if p.sink != nil {
		p.sink.Add(record)
	}

	return &ProcessResult{Record: record}
}

// SetSourceName updates the default source name for untagged lines.
func (p *PassthroughProcessor) SetSourceName(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sourceName = name
}

func (p *PassthroughProcessor) getSourceName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sourceName
}
