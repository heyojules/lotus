package ingest

import (
	"testing"

	"github.com/control-theory/lotus/internal/model"
)

type recordingSink struct {
	records []*model.LogRecord
}

func (s *recordingSink) Add(record *model.LogRecord) {
	s.records = append(s.records, record)
}

func TestNewEnvelopeProcessor_DefaultParse(t *testing.T) {
	t.Parallel()

	p, err := NewEnvelopeProcessor("", nil, "")
	if err != nil {
		t.Fatalf("NewEnvelopeProcessor returned error: %v", err)
	}
	if p.Name() != ProcessorModeParse {
		t.Fatalf("processor name = %q, want %q", p.Name(), ProcessorModeParse)
	}
	if _, ok := p.(*Processor); !ok {
		t.Fatalf("processor type = %T, want *Processor", p)
	}
}

func TestNewEnvelopeProcessor_Passthrough(t *testing.T) {
	t.Parallel()

	p, err := NewEnvelopeProcessor("passthrough", nil, "")
	if err != nil {
		t.Fatalf("NewEnvelopeProcessor returned error: %v", err)
	}
	if p.Name() != ProcessorModePassthrough {
		t.Fatalf("processor name = %q, want %q", p.Name(), ProcessorModePassthrough)
	}
	if _, ok := p.(*PassthroughProcessor); !ok {
		t.Fatalf("processor type = %T, want *PassthroughProcessor", p)
	}
}

func TestNewEnvelopeProcessor_InvalidMode(t *testing.T) {
	t.Parallel()

	if _, err := NewEnvelopeProcessor("unknown", nil, ""); err == nil {
		t.Fatal("expected error for invalid processor mode")
	}
}

func TestPassthroughProcessor_ProcessEnvelope_UsesDefaultSource(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{}
	p := NewPassthroughProcessor(sink, "stdin")

	result := p.ProcessEnvelope(model.IngestEnvelope{Line: "hello world"})
	if result == nil || result.Record == nil {
		t.Fatal("expected non-nil process result")
	}

	if got := len(sink.records); got != 1 {
		t.Fatalf("sink records = %d, want 1", got)
	}

	record := sink.records[0]
	if record.Source != "stdin" {
		t.Fatalf("record source = %q, want %q", record.Source, "stdin")
	}
	if record.Message != "hello world" {
		t.Fatalf("record message = %q, want %q", record.Message, "hello world")
	}
}

func TestPassthroughProcessor_ProcessEnvelope_SourceOverride(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{}
	p := NewPassthroughProcessor(sink, "stdin")

	result := p.ProcessEnvelope(model.IngestEnvelope{Source: "tcp", Line: "hello"})
	if result == nil || result.Record == nil {
		t.Fatal("expected non-nil process result")
	}

	record := sink.records[0]
	if record.Source != "tcp" {
		t.Fatalf("record source = %q, want %q", record.Source, "tcp")
	}
}
