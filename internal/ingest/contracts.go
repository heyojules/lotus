package ingest

import "github.com/tinytelemetry/lotus/internal/model"

const (
	// ProcessorNameOTEL is the single processor implementation name.
	ProcessorNameOTEL = "otel"
)

// EnvelopeProcessor consumes source-tagged ingest lines and emits canonical records.
type EnvelopeProcessor interface {
	Name() string
	ProcessEnvelope(model.IngestEnvelope) *ProcessResult
}

// NewEnvelopeProcessor creates the OTEL processor implementation.
func NewEnvelopeProcessor(sink RecordSink, sourceName string) EnvelopeProcessor {
	return NewProcessor(sink, sourceName)
}
