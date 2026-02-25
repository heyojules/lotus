package ingest

import (
	"fmt"
	"strings"

	"github.com/control-theory/lotus/internal/model"
)

const (
	// ProcessorModeParse parses/normalizes JSON and text logs into canonical records.
	ProcessorModeParse = "parse"
	// ProcessorModePassthrough keeps processing lightweight and skips JSON parsing.
	ProcessorModePassthrough = "passthrough"
)

// EnvelopeProcessor consumes source-tagged ingest lines and emits canonical records.
type EnvelopeProcessor interface {
	Name() string
	ProcessEnvelope(model.IngestEnvelope) *ProcessResult
}

// NewEnvelopeProcessor creates a processor implementation for the requested mode.
// Empty mode defaults to parse for backward compatibility.
func NewEnvelopeProcessor(mode string, sink RecordSink, sourceName string) (EnvelopeProcessor, error) {
	switch normalizeMode(mode) {
	case ProcessorModeParse:
		return NewProcessor(sink, sourceName), nil
	case ProcessorModePassthrough:
		return NewPassthroughProcessor(sink, sourceName), nil
	default:
		return nil, fmt.Errorf("unknown processor mode %q", mode)
	}
}

// IsValidProcessorMode reports whether mode is supported.
func IsValidProcessorMode(mode string) bool {
	switch normalizeMode(mode) {
	case ProcessorModeParse, ProcessorModePassthrough:
		return true
	default:
		return false
	}
}

func normalizeMode(mode string) string {
	if strings.TrimSpace(mode) == "" {
		return ProcessorModeParse
	}
	return strings.ToLower(strings.TrimSpace(mode))
}
