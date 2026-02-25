package main

import (
	"context"
	"testing"
	"time"

	"github.com/tinytelemetry/lotus/internal/ingest"
	"github.com/tinytelemetry/lotus/internal/model"
)

type fakeSource struct {
	name    string
	lines   chan model.IngestEnvelope
	stopped chan struct{}
}

func newFakeSource(name string, buffer int) *fakeSource {
	return &fakeSource{
		name:    name,
		lines:   make(chan model.IngestEnvelope, buffer),
		stopped: make(chan struct{}),
	}
}

func (s *fakeSource) Lines() <-chan model.IngestEnvelope { return s.lines }
func (s *fakeSource) Name() string                       { return s.name }

func (s *fakeSource) Stop() {
	select {
	case <-s.stopped:
		return
	default:
		close(s.stopped)
		close(s.lines)
	}
}

func TestSourceMultiplexer_ForwardsFromAllSources(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := newFakeSource("a", 2)
	b := newFakeSource("b", 2)

	mux := NewSourceMultiplexer(ctx, []NamedLogSource{a, b}, 16)
	mux.Start()
	defer mux.Stop()

	a.lines <- model.IngestEnvelope{Source: "a", Line: "alpha"}
	b.lines <- model.IngestEnvelope{Source: "b", Line: "beta"}
	a.Stop()
	b.Stop()

	got := map[string]bool{}
	timeout := time.After(2 * time.Second)
	for len(got) < 2 {
		select {
		case env, ok := <-mux.Lines():
			if !ok {
				t.Fatalf("multiplexer closed before receiving expected lines: %+v", got)
			}
			got[env.Line] = true
		case <-timeout:
			t.Fatalf("timed out waiting for multiplexed lines: %+v", got)
		}
	}

	if !got["alpha"] || !got["beta"] {
		t.Fatalf("missing expected lines: %+v", got)
	}
}

func TestSourceMultiplexer_StopInvokesSourceStop(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	src := newFakeSource("x", 1)
	mux := NewSourceMultiplexer(ctx, []NamedLogSource{src}, 8)
	mux.Start()

	mux.Stop()

	select {
	case <-src.stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("expected source Stop() to be called")
	}
}

type integrationSink struct {
	records []*model.LogRecord
}

func (s *integrationSink) Add(record *model.LogRecord) {
	s.records = append(s.records, record)
}

func TestPipelineIntegration_OTEL_MultiSourceFlow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tcp := newFakeSource("tcp-source", 4)
	stdin := newFakeSource("stdin-source", 4)

	mux := NewSourceMultiplexer(ctx, []NamedLogSource{tcp, stdin}, 16)
	mux.Start()
	defer mux.Stop()

	sink := &integrationSink{}
	processor := ingest.NewEnvelopeProcessor(sink, "")

	tcp.lines <- model.IngestEnvelope{
		Source: "tcp",
		Line:   `{"timeUnixNano":"1761238800000000000","severityText":"Info","body":{"stringValue":"json from tcp"},"attributes":[{"key":"service.name","value":{"stringValue":"payments"}}]}`,
	}
	stdin.lines <- model.IngestEnvelope{
		Source: "stdin",
		Line:   `{"timeUnixNano":"1761238801000000000","severityText":"Error","body":{"stringValue":"otel from stdin"},"attributes":[{"key":"service.name","value":{"stringValue":"stdin-worker"}}]}`,
	}
	tcp.Stop()
	stdin.Stop()

	for env := range mux.Lines() {
		processor.ProcessEnvelope(env)
	}

	if len(sink.records) != 2 {
		t.Fatalf("records = %d, want 2", len(sink.records))
	}

	bySource := map[string]*model.LogRecord{}
	for _, rec := range sink.records {
		bySource[rec.Source] = rec
	}

	tcpRecord := bySource["tcp"]
	if tcpRecord == nil {
		t.Fatal("missing tcp record")
	}
	if tcpRecord.Message != "json from tcp" {
		t.Fatalf("tcp message = %q, want %q", tcpRecord.Message, "json from tcp")
	}
	if tcpRecord.Service != "payments" {
		t.Fatalf("tcp service = %q, want %q", tcpRecord.Service, "payments")
	}

	stdinRecord := bySource["stdin"]
	if stdinRecord == nil {
		t.Fatal("missing stdin record")
	}
	if stdinRecord.Level != "ERROR" {
		t.Fatalf("stdin level = %q, want %q", stdinRecord.Level, "ERROR")
	}
	if stdinRecord.Message != "otel from stdin" {
		t.Fatalf("stdin message = %q, want %q", stdinRecord.Message, "otel from stdin")
	}
}

func TestPipelineIntegration_OTEL_DropsNonOTELPayload(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	src := newFakeSource("tcp-source", 2)
	mux := NewSourceMultiplexer(ctx, []NamedLogSource{src}, 8)
	mux.Start()
	defer mux.Stop()

	sink := &integrationSink{}
	processor := ingest.NewEnvelopeProcessor(sink, "")

	raw := `{"level":30,"msg":"json message","service":"orders"}`
	src.lines <- model.IngestEnvelope{Source: "tcp", Line: raw}
	src.Stop()

	for env := range mux.Lines() {
		processor.ProcessEnvelope(env)
	}

	if len(sink.records) != 0 {
		t.Fatalf("records = %d, want 0", len(sink.records))
	}
}
