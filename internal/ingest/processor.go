package ingest

import (
	"strings"

	"github.com/control-theory/lotus/internal/duckdb"
	"github.com/control-theory/lotus/internal/model"
)

// Processor handles log line parsing, analysis, and routing to storage.
type Processor struct {
	insertBuffer *duckdb.InsertBuffer
	sourceName   string

	// JSON accumulation for multi-line JSON support
	jsonBuffer   strings.Builder
	jsonDepth    int
	inJsonObject bool

	// Result from processCompleteJSON, consumed by ProcessLine
	lastResult *ProcessResult
}

// NewProcessor creates a new log processor.
func NewProcessor(
	insertBuffer *duckdb.InsertBuffer,
	sourceName string,
) *Processor {
	return &Processor{
		insertBuffer: insertBuffer,
		sourceName:   sourceName,
	}
}

// ProcessResult holds the result of processing a log line.
type ProcessResult struct {
	Record *model.LogRecord
}

// ProcessLine processes a single log line, returning the parsed entry.
// Returns nil if the line is being accumulated as part of a multi-line JSON object.
func (p *Processor) ProcessLine(line string) *ProcessResult {
	// Handle multi-line JSON accumulation
	if p.tryAccumulateJSON(line) {
		// If accumulation completed a JSON object, return its result
		if p.lastResult != nil {
			result := p.lastResult
			p.lastResult = nil
			return result
		}
		return nil
	}

	return p.processEntry(line)
}

// processEntry parses a line, analyzes it, and stores it.
func (p *Processor) processEntry(line string) *ProcessResult {
	// Try parsing as JSON first, fall back to plain text
	record := ParseJSONLogEntry(line)
	if record == nil {
		record = CreateFallbackLogEntry(line)
	}

	// Fill in fields derived by the processor
	record.Service = ExtractService(record.Attributes)
	record.Hostname = record.Attributes["host"]
	record.Source = p.sourceName

	// Insert into DuckDB via buffer
	if p.insertBuffer != nil {
		p.insertBuffer.Add(record)
	}

	return &ProcessResult{
		Record: record,
	}
}

// tryAccumulateJSON attempts to accumulate multi-line JSON and process when complete.
// Returns true if the line was consumed (either accumulated or completed).
func (p *Processor) tryAccumulateJSON(line string) bool {
	trimmed := strings.TrimSpace(line)

	if !p.inJsonObject {
		if trimmed == "{" || strings.HasPrefix(trimmed, "{") {
			p.inJsonObject = true
			p.jsonBuffer.Reset()
			p.jsonDepth = 0
			p.jsonBuffer.WriteString(line)
			p.jsonBuffer.WriteString("\n")

			p.jsonDepth += CountJSONDepth(line)

			if p.jsonDepth <= 0 {
				completeJSON := strings.TrimSpace(p.jsonBuffer.String())
				p.resetJSONAccumulation()
				p.processCompleteJSON(completeJSON)
				return true
			}

			return true
		}
		return false
	}

	p.jsonBuffer.WriteString(line)
	p.jsonBuffer.WriteString("\n")
	p.jsonDepth += CountJSONDepth(line)

	if p.jsonDepth <= 0 {
		completeJSON := strings.TrimSpace(p.jsonBuffer.String())
		p.resetJSONAccumulation()
		p.processCompleteJSON(completeJSON)
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
	p.jsonBuffer.Reset()
}

// processCompleteJSON processes a complete JSON object (single or multi-line).
func (p *Processor) processCompleteJSON(jsonStr string) {
	// This goes through the same path as a single line
	p.lastResult = p.processEntry(jsonStr)
}

// SetSourceName updates the source name used for log records.
func (p *Processor) SetSourceName(name string) {
	p.sourceName = name
}
