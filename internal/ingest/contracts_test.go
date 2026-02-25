package ingest

import (
	"testing"

	"github.com/tinytelemetry/lotus/internal/model"
)

type recordingSink struct {
	records []*model.LogRecord
}

func (s *recordingSink) Add(record *model.LogRecord) {
	s.records = append(s.records, record)
}

func TestNewEnvelopeProcessor_DefaultOTEL(t *testing.T) {
	t.Parallel()

	p := NewEnvelopeProcessor(nil, "")
	if p.Name() != ProcessorNameOTEL {
		t.Fatalf("processor name = %q, want %q", p.Name(), ProcessorNameOTEL)
	}
	if _, ok := p.(*Processor); !ok {
		t.Fatalf("processor type = %T, want *Processor", p)
	}
}

func TestProcessor_ProcessEnvelope_OTELBatch(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{}
	p := NewProcessor(sink, "stdin")

	line := `{
		"resourceLogs": [
			{
				"resource": {
					"attributes": [
						{"key":"service.name","value":{"stringValue":"api"}}
					]
				},
				"scopeLogs": [
					{
						"logRecords": [
							{"timeUnixNano":"1739876543210000000","severityText":"Info","body":{"stringValue":"log one"}},
							{"timeUnixNano":"1739876544210000000","severityText":"Warn","body":{"stringValue":"log two"}}
						]
					}
				]
			}
		]
	}`

	result := p.ProcessEnvelope(model.IngestEnvelope{Source: "tcp", Line: line})
	if result == nil || result.Record == nil {
		t.Fatal("expected non-nil process result")
	}

	if got := len(sink.records); got != 2 {
		t.Fatalf("sink records = %d, want 2", got)
	}

	if sink.records[0].Message != "log one" {
		t.Fatalf("first message = %q, want %q", sink.records[0].Message, "log one")
	}
	if sink.records[1].Message != "log two" {
		t.Fatalf("second message = %q, want %q", sink.records[1].Message, "log two")
	}
	if sink.records[0].Source != "tcp" || sink.records[1].Source != "tcp" {
		t.Fatalf("all records should inherit source %q", "tcp")
	}
	if sink.records[0].Service != "api" || sink.records[1].Service != "api" {
		t.Fatalf("all records should inherit service %q", "api")
	}
}

func TestProcessor_ProcessEnvelope_NonOTELDropped(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{}
	p := NewProcessor(sink, "stdin")

	result := p.ProcessEnvelope(model.IngestEnvelope{Source: "tcp", Line: `{"level":"info","msg":"legacy"}`})
	if result != nil {
		t.Fatal("expected nil result for non-OTEL JSON")
	}
	if len(sink.records) != 0 {
		t.Fatalf("expected zero sink records, got %d", len(sink.records))
	}
}
