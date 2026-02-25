# Layer Interface Contracts

This document captures current boundaries and the minimal target shape for swappable layers.

## Current Contracts

### Ingestion -> Processing

- Transport: `<-chan model.IngestEnvelope`
- Producer: `NamedLogSource` via `SourceMultiplexer`
- Consumer: `ingest.EnvelopeProcessor.ProcessEnvelope(env model.IngestEnvelope)`

### Processing -> Storage

- Type: `*model.LogRecord`
- Method: `InsertBuffer.Add(record)`
- Semantics: enqueue for async batch flush; should not block on DB IO

### Storage -> Read Surface

- Server-side contract: `model.ReadAPI`
- Socket dispatch surface: `model.LogQuerier` methods
- Ad hoc SQL: `SchemaQuerier.ExecuteQuery` (read-only guarded)

## Behavioral Contracts (Important)

1. Ingestion channels may block under sustained overload; buffers provide absorption, not infinite capacity.
2. Insert buffer guarantees eventual flush on graceful stop (`InsertBuffer.Stop()`).
3. Read surfaces are logically read-only and should not mutate storage state.
4. `model.LogRecord` is the canonical cross-layer data model.

## Implemented Boundary Upgrades

### Contract 1: Source Envelope

```go
type IngestEnvelope struct {
  Source     string
  Line       string
}
```

Benefits:

- Accurate multi-source attribution per record.
- Keeps plugin architecture simple and explicit.

### Contract 2: Record Sink Port

```go
type RecordSink interface {
  Add(*model.LogRecord)
}
```

Benefits:

- Processor no longer depends concretely on DuckDB insert buffer.
- Enables isolated processor tests with fake sinks.

### Contract 3: Read API Split

```go
type ReadAPI interface {
  model.LogQuerier
  model.SchemaQuerier
}
```

Benefits:

- One explicit read contract for HTTP and socket.
- Prevents protocol-specific business logic drift.

## Implemented Order

1. Introduced `IngestEnvelope`.
2. Added `RecordSink` in processing.
3. Standardized read-surface server contracts on `ReadAPI`.

## Optional Later

1. Add timestamp to envelope if latency instrumentation is needed.
2. Add protocol versioning (`v1`) if external integrations expand.
