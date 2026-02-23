# Layer 4: Read Surface

## Purpose

Expose stored data for humans (TUI) and agents/tools (HTTP API) without exposing write access.

## Owned Components

- `internal/httpserver/server.go`
- `internal/socketrpc/server.go`
- `internal/socketrpc/client.go`
- `internal/tui/*`
- `cmd/lotus-cli/*`

## Current Design

There are two read surfaces:

1. HTTP API (`/api/health`, `/api/schema`, `/api/query`) served by `internal/httpserver`.
2. Unix socket JSON-RPC used by `lotus-cli` TUI (`internal/socketrpc` + `internal/tui`).

Both surfaces ultimately depend on storage-layer interfaces:

- HTTP: `QueryStore` (`model.ReadAPI`)
- Socket server: `model.ReadAPI` (dispatch currently uses `LogQuerier` methods)

## Why It Is Decoupled

- Service process can run headless without TUI.
- TUI is a separate binary connecting over socket.
- Agent-facing API is protocol-agnostic from TUI concerns.

## Current Strengths

- Strong separation between service runtime and terminal UI.
- Read-only HTTP behavior by design.
- Narrow interface for HTTP surface reduces accidental coupling.
- Socket RPC keeps TUI responsive without embedding DB logic in CLI.

## Current Friction

- API and socket expose overlapping capabilities through different protocols.
- Socket method dispatch is string-based and manually mapped.
- No formal versioned contract for external integrators.

## Implemented Boundary Upgrade

Both protocols are standardized on one shared read contract:

- `ReadAPI = model.LogQuerier + model.SchemaQuerier`
- Keep manual HTTP handlers and manual socket dispatch for now.

This avoids drift without introducing generators or framework complexity.

## Optional Later

1. Add explicit `v1` versioning when there is external client churn.
2. Move socket dispatch to table-driven registration only when switch growth hurts maintainability.
3. Add typed endpoints for top queries only when usage patterns stabilize.
