package ingest

import (
	"log"
	"strings"
	"sync"

	"github.com/tinytelemetry/lotus/internal/model"
)

// maxJSONBufferSize is the maximum size of accumulated multi-line JSON before
// the buffer is reset to prevent OOM from malformed input with unclosed braces.
const maxJSONBufferSize = 10 * 1024 * 1024 // 10 MB

// Processor handles log line parsing, analysis, and routing to storage.
// All methods are safe for concurrent use.
type Processor struct {
	mu         sync.Mutex
	sink       RecordSink
	sourceName string

	// JSON accumulation for multi-line JSON support
	jsonBuffer   strings.Builder
	jsonDepth    int
	inJsonObject bool
	jsonSource   string

	// Result from processCompleteJSON, consumed by ProcessLine
	lastResult *ProcessResult
}

func (p *Processor) Name() string { return ProcessorNameOTEL }

// RecordSink accepts processed records.
// InsertBuffer is one implementation.
type RecordSink interface {
	Add(*model.LogRecord)
}

// NewProcessor creates a new log processor.
func NewProcessor(
	sink RecordSink,
	sourceName string,
) *Processor {
	return &Processor{
		sink:       sink,
		sourceName: sourceName,
	}
}

// ProcessResult holds the result of processing a log line.
type ProcessResult struct {
	Record *model.LogRecord
}

// ProcessLine processes a single log line, returning the parsed entry.
// Returns nil if the line is being accumulated as part of a multi-line JSON object.
// Safe for concurrent use.
func (p *Processor) ProcessLine(line string) *ProcessResult {
	return p.ProcessEnvelope(model.IngestEnvelope{
		Source: p.sourceName,
		Line:   line,
	})
}

// ProcessEnvelope processes one source-tagged line and returns the parsed entry.
// Returns nil if the line is being accumulated as part of a multi-line JSON object.
// Safe for concurrent use.
func (p *Processor) ProcessEnvelope(env model.IngestEnvelope) *ProcessResult {
	p.mu.Lock()
	defer p.mu.Unlock()

	if env.Line == "" {
		return nil
	}

	source := env.Source
	if source == "" {
		source = p.sourceName
	}

	// Handle multi-line JSON accumulation
	if p.tryAccumulateJSON(env.Line, source) {
		// If accumulation completed a JSON object, return its result
		if p.lastResult != nil {
			result := p.lastResult
			p.lastResult = nil
			return result
		}
		return nil
	}

	return p.processEntry(env.Line, source)
}

// processEntry parses an OTEL line, enriches it, and stores it.
// Caller must hold p.mu. The lock is released before calling insertBuffer.Add()
// to avoid holding the mutex during potential backpressure-induced DuckDB flushes.
func (p *Processor) processEntry(line, source string) *ProcessResult {
	// Parse-mode accepts OTEL JSON only.
	records := ParseJSONLogEntries(line)
	if len(records) == 0 {
		return nil
	}

	for _, record := range records {
		// Fill in fields derived by the processor.
		record.Service = ExtractService(record.Attributes)
		if record.Service == "unknown" && record.App != "" && record.App != "default" {
			record.Service = record.App
		}
		record.Hostname = record.Attributes["host"]
		if record.Hostname == "" {
			record.Hostname = record.Attributes["hostname"]
		}
		if record.Hostname == "" {
			record.Hostname = record.Attributes["host.name"]
		}
		record.Source = source
	}

	sink := p.sink
	// Release lock before potentially slow buffer insertion.
	p.mu.Unlock()

	if sink != nil {
		for _, record := range records {
			sink.Add(record)
		}
	}

	// Re-acquire lock (caller expects it held via defer).
	p.mu.Lock()

	return &ProcessResult{
		Record: records[0],
	}
}

// tryAccumulateJSON attempts to accumulate multi-line JSON and process when complete.
// Returns true if the line was consumed (either accumulated or completed).
func (p *Processor) tryAccumulateJSON(line, source string) bool {
	trimmed := strings.TrimSpace(line)

	if !p.inJsonObject {
		if trimmed == "{" || strings.HasPrefix(trimmed, "{") {
			p.inJsonObject = true
			p.jsonBuffer.Reset()
			p.jsonDepth = 0
			p.jsonSource = source
			p.jsonBuffer.WriteString(line)
			p.jsonBuffer.WriteString("\n")

			p.jsonDepth += CountJSONDepth(line)

			if p.jsonDepth <= 0 {
				completeJSON := strings.TrimSpace(p.jsonBuffer.String())
				jsonSource := p.jsonSource
				p.resetJSONAccumulation()
				p.processCompleteJSON(completeJSON, jsonSource)
				return true
			}

			return true
		}
		return false
	}

	p.jsonBuffer.WriteString(line)
	p.jsonBuffer.WriteString("\n")

	if p.jsonBuffer.Len() > maxJSONBufferSize {
		log.Printf("ingest: multi-line JSON buffer exceeded %d bytes, resetting", maxJSONBufferSize)
		p.resetJSONAccumulation()
		return false
	}

	p.jsonDepth += CountJSONDepth(line)

	if p.jsonDepth <= 0 {
		completeJSON := strings.TrimSpace(p.jsonBuffer.String())
		jsonSource := p.jsonSource
		p.resetJSONAccumulation()
		p.processCompleteJSON(completeJSON, jsonSource)
		return true
	}

	return true
}

// CountJSONDepth counts the net change in JSON nesting depth for a line.
func CountJSONDepth(line string) int {
	depth := 0
	inString := false
	escaped := false

	for _, char := range line {
		if escaped {
			escaped = false
			continue
		}

		switch char {
		case '\\':
			if inString {
				escaped = true
			}
		case '"':
			inString = !inString
		case '{', '[':
			if !inString {
				depth++
			}
		case '}', ']':
			if !inString {
				depth--
			}
		}
	}

	return depth
}

// resetJSONAccumulation resets the JSON accumulation state.
func (p *Processor) resetJSONAccumulation() {
	p.inJsonObject = false
	p.jsonDepth = 0
	p.jsonSource = ""
	p.jsonBuffer.Reset()
}

// processCompleteJSON processes a complete JSON object (single or multi-line).
func (p *Processor) processCompleteJSON(jsonStr, source string) {
	// This goes through the same path as a single line
	p.lastResult = p.processEntry(jsonStr, source)
}

// SetSourceName updates the source name used for log records.
// Safe for concurrent use.
func (p *Processor) SetSourceName(name string) {
	p.mu.Lock()
	p.sourceName = name
	p.mu.Unlock()
}
