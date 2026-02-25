# Layer 3: Storage

## Purpose

Persist canonical logs and provide all read/query primitives from DuckDB.

## Owned Components

- `internal/duckdb/store.go`
- `internal/duckdb/insert.go`
- `internal/duckdb/queries.go`
- `internal/duckdb/retention.go`
- `internal/duckdb/migrate/*`

## Current Design

`duckdb.Store` owns DB lifecycle, migrations, query timeout, and query methods.

Write path:

- `InsertBuffer.Add()` appends to pending batch.
- Flush triggers on size (`insert-batch-size`, default 2000) or interval (`insert-flush-interval`, default 100ms).
- Worker calls `Store.InsertLogBatch()` transactionally.

Read path:

- `Store` implements `model.LogQuerier` and `model.SchemaQuerier`.
- HTTP and socket layers read through those interfaces.

Retention:

- Optional hourly cleanup deletes logs older than `log-retention` days.
- Optional periodic backups create local DuckDB snapshots and can upload to S3-compatible storage.

## Why It Is Decoupled

- DuckDB is the single source of truth; no secondary in-memory read models.
- Query contracts are expressed in interfaces in `internal/model/iface.go`.
- Read surfaces do not know SQL internals beyond query contracts.

## Current Strengths

- Clear append-only write flow with asynchronous batching.
- Read-only query guardrails in `ExecuteQuery` (SELECT/WITH only + keyword checks).
- Migration-first startup ensures schema consistency.
- Supports app-scoped queries without duplicating storage.

## Current Friction

- Single `Store` type carries both write and read responsibilities.
- Coarse mutexing may limit concurrency under high mixed read/write load.
- `ExecuteQuery` returns map rows with dynamic columns; weak type guarantees for consumers.

## Implemented Boundary Upgrade

Keep one `Store` implementation and use logical boundary interfaces:

```go
type LogWriter interface {
  InsertLogBatch(records []*model.LogRecord) error
}

type LogReader interface {
  model.LogQuerier
  model.SchemaQuerier
}
```

Write-side and read-side are represented as separate contracts while still using one store implementation.

This keeps implementation simple but prevents cross-layer leakage.

## Optional Later

1. Add query telemetry (duration, rows, timeout count) when performance tuning starts.
2. Add typed read endpoints for common queries only when `/api/query` becomes operationally risky.
3. Add restore tooling and upload retry queue for backup workflow.
