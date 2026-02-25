<p align="center">
  <img src="assets/26e6c9d4-1e0a-42c9-a7b2-cef549bf96db-1.png" alt="Lotus Logo" width="280" />
</p>

<h1 align="center">Lotus</h1>
<p align="center"><strong>Tiny telemetry tool built on standards for agents, from first principles</strong></p>

<p align="center">
  Imperative for developer, declarative for agent.
</p>

> [!WARNING]
> **Still in heavy development** — not ready for production use.

## Why Lotus exists

Lotus is a thin observability layer that ingests logs, stores them in DuckDB, and exposes a read-only HTTP API queryable by AI agents and scripts. A TUI dashboard (`lotus-tui`) is included for humans.


**Design principles:**

- **Standards first** — OTEL log data model by default, no custom vendor schema
- **Zero friction** — drop the binary, pipe your output, done
- **Agent-first** — the HTTP API is the primary read surface, designed for autonomous programmatic access
- **Minimal by intent** — keep only essential ingestion, storage, and query surfaces
- **No noise** — no heavy platform layers, no observability-suite sprawl
- **Single source of truth** — all state lives in DuckDB, no caches, no secondary stores
- **Thin by design** — easy to extend with new inputs or read surfaces because there's almost nothing in the way

## Architecture

```
Input Plugins          Processing              Storage              Read Surfaces
─────────────       ──────────────          ─────────────       ─────────────────

  TCP:4000  ──┐     ┌─ Processor ─┐        ┌───────────┐        ┌─ HTTP API
              ├──→  │  parse +    │──→     │  DuckDB   │──→     │  (agents, scripts)
  stdin     ──┘     │  OTEL logs  │        │           │        └──────────────
     ↑              └─────────────┘        │           │        ┌─ Socket RPC
     │                    ↓                │           │──→     │  (lotus-tui)
  SourceMux          InsertBuffer          │           │        └──────────────
                    (batch append)         └───────────┘
```

## Getting started

**Same machine** — pipe stdout/stderr directly:

```sh
your-app 2>&1 | lotus
```

**Other machines** — send newline-delimited OTEL log JSON to TCP port `4000`:

```yaml
host: 0.0.0.0
tcp-port: 4000
```

Single OTEL log-record example:

```json
{"timeUnixNano":"1761238800000000000","severityText":"Info","body":{"stringValue":"payment created"},"attributes":[{"key":"service.name","value":{"stringValue":"billing-api"}}]}
```

> [!NOTE]
> TCP ingest currently has no built-in TLS/auth. Expose only on trusted networks or behind a secure tunnel.

For production durable forwarding via rsyslog, see [`docs/operations/rsyslog-forwarder.md`](docs/operations/rsyslog-forwarder.md).

## Themes

Lotus ships with 12 color themes:

```yaml
skin: dracula
```

Available: `lotus-dark`, `lotus-light`, `dracula`, `github-light`, `gruvbox`, `matrix`, `monokai`, `nord`, `solarized-dark`, `solarized-light`, `spring`, `vs-code-light`.

## Screenshots

<p align="center">
  <img src="assets/Screenshot 2026-02-23 at 22.22.09.png" alt="Lotus running" width="500" />
</p>

<p align="center">
  <img src="assets/Screenshot 2026-02-23 at 10.25.45.png" alt="Lotus TUI dashboard" width="700" />
</p>

## Further reading

- [`docs/layers/README.md`](docs/layers/README.md) — layer docs and interface contracts
- [`docs/operations/rsyslog-forwarder.md`](docs/operations/rsyslog-forwarder.md) — production forwarding
- [`docs/operations/duckdb-backups.md`](docs/operations/duckdb-backups.md) — backup strategy

## License

See [LICENSE](LICENSE) for details.
