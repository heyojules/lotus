# Layer 2: Processing Pipeline

## Purpose

Transform raw lines into canonical `model.LogRecord` and forward to storage.

## Owned Components

- `internal/ingest/processor.go`
- `internal/ingest/extractor.go`
- `internal/logparse/*`
- `internal/timestamp/*`

## Current Design

Processing runs behind a swappable processor contract:

```go
type EnvelopeProcessor interface {
  Name() string
  ProcessEnvelope(model.IngestEnvelope) *ProcessResult
}
```

Current implementation:

- `otel` (`Processor`) for OTEL parse + normalize behavior.

`otel` does three jobs:

1. Multi-line JSON accumulation (`tryAccumulateJSON`, `CountJSONDepth`)
2. Parsing and normalization (`ParseJSONLogEntries`) for OTEL log model payloads
3. Storage handoff (`insertBuffer.Add(record)`)

Main output type:

```go
type ProcessResult struct {
  Record *model.LogRecord
}
```

Canonical record type is shared across layers in `internal/model/types.go`.

## Why It Is Decoupled

- Parsing logic is isolated from input plugin details.
- Storage is accessed through insert buffer, not direct DB writes.
- Output model is stable (`model.LogRecord`) and reused by query/read paths.

## Current Strengths

- OTEL-first processing path with deterministic behavior.
- Handles both OTEL single-record and OTEL export-envelope shapes.
- Includes bounded multi-line JSON buffer (10 MB cap) to avoid unbounded growth.

## Current Friction

- `Processor` mixes pure transformation and side effects (DB enqueue).
- Lock scope is complex due to JSON state plus async insertion handoff.
- Error outcomes are mostly implicit (nil result/log output) rather than typed.

## Implemented Boundary Upgrade

One narrow dependency inversion is in place:

```go
type RecordSink interface {
  Add(*model.LogRecord)
}
```

Processors depend on this interface, with `InsertBuffer` as one implementation.

This is enough to keep business logic swappable without adding architecture layers.

## Optional Later

1. Add typed status codes if observability/debugging needs them:

```go
type ProcessStatus string
const (
  Processed ProcessStatus = "processed"
  Skipped   ProcessStatus = "skipped"
  Invalid   ProcessStatus = "invalid"
)
```

2. Split parser/normalizer/sink into separate components only if the processor grows too large.
