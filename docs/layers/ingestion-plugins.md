# Layer 1: Ingestion Plugins

## Purpose

Accept log lines from external inputs and expose them as a unified stream.

## Owned Components

- `cmd/lotus/input_plugins.go`
- `cmd/lotus/source_mux.go`
- `internal/logsource/logsource.go`
- `internal/logsource/stdin.go`
- `internal/logsource/tcp.go`
- `internal/tcpserver/server.go`

## Current Design

`InputSourcePlugin` is the runtime plugin primitive:

```go
type InputSourcePlugin interface {
  Name() string
  Enabled() bool
  Build(ctx context.Context) (NamedLogSource, error)
}
```

`NamedLogSource` is an alias for `logsource.LogSource`:

```go
type LogSource interface {
  Lines() <-chan model.IngestEnvelope
  Stop()
  Name() string
}
```

Enabled plugins are built and then merged through `SourceMultiplexer` into one buffered channel (`DefaultMuxBuffer = 50_000`).

## Why It Is Decoupled

- Input format details (TCP, stdin) are isolated from parsing/storage.
- Downstream consumes `IngestEnvelope` from a single channel.
- Plugins can be added without touching storage or read surfaces.

## Current Strengths

- Very small plugin API.
- Good fan-in behavior via dedicated goroutine per source.
- Explicit source lifecycle (`Stop`) and cancellation handling.
- Sensible buffering in source and multiplexer.

## Current Friction

- Source metadata is lossy at merge time. Processing gets a single `sourceName` (primary source), not per-line source identity.
- `Stop()` does not return errors, so shutdown failures are silent.
- Health and backpressure signals are not surfaced as first-class metrics/events.

## Implemented Boundary Upgrade

Plugin model stays the same, and the transport boundary is now upgraded:

```go
type IngestEnvelope struct {
  Source     string
  Line       string
}
```

Channel type is now `<-chan IngestEnvelope`.

This keeps the architecture simple and makes sources truly swappable and composable.

## Optional Later

1. Add `ReceivedAt` if needed for latency analysis.
2. Change `Stop()` to `Stop(ctx) error` if shutdown failure handling becomes important.
3. Add counters only if operating at scale requires them.
