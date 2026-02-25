# Lotus Layered Design Docs

These docs describe Lotus as four decoupled layers:

1. Ingestion plugins
2. Processing pipeline
3. Storage
4. Read surfaces

They are written with a simplicity-first rule:

- Clear boundaries
- Small interfaces
- Swappable implementations
- No framework-heavy abstractions

## Status

- Implemented architecture: documented under each file's `Current Design` section.
- Implemented boundary upgrades: documented under `Implemented Boundary Upgrade`.
- Future ideas only: documented under `Optional Later`.

## Runtime Flow

```mermaid
flowchart LR
  A["Ingestion Plugins\ncmd/lotus/input_plugins.go"] --> B["Source Multiplexer\ncmd/lotus/source_mux.go"]
  B --> C["Processing Pipeline\ninternal/ingest/processor.go"]
  C --> D["Insert Buffer\ninternal/duckdb/insert.go"]
  D --> E["DuckDB Store\ninternal/duckdb/store.go"]
  E --> F["HTTP API\ninternal/httpserver/server.go"]
  E --> G["Socket RPC\ninternal/socketrpc/server.go"]
  G --> H["TUI Client\ncmd/lotus-tui/main.go"]
```

## Files

- [Ingestion Plugins](./ingestion-plugins.md)
- [Processing Pipeline](./processing-pipeline.md)
- [Storage](./storage.md)
- [Read Surface](./read-surface.md)
- [Interface Contracts](./interfaces.md)

## Operations

- [Durable Local Forwarding With rsyslog](../operations/rsyslog-forwarder.md)

## Decision Rule

For architecture changes, apply this filter:

1. Does this make a layer easier to swap?
2. Does this reduce coupling at a boundary?
3. Is it implementable with a tiny interface and direct wiring?

If any answer is no, defer it.
