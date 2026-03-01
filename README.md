<p align="center">
  <img src="assets/26e6c9d4-1e0a-42c9-a7b2-cef549bf96db-1.png" alt="Lotus Logo" width="280" />
</p>

<h1 align="center">Lotus</h1>
<p align="center"><strong>Tiny, standards-based telemetry tool, built from first principles</strong></p>

<p align="center">
  <em>"Imperative for user, declarative for agent."</em>
</p>

> [!WARNING]
> **Still in heavy development** — not ready for production use.

## Why Lotus exists

Lotus is a thin layer that ingests telemetry/analytics, stores them in DuckDB, and exposes a read-only HTTP API queryable by AI agents and scripts. A TUI dashboard (`lotus-tui`) is included for humans, organized into pages (Logs, Metrics, Analytics) with switchable views within each page.


**Design principles:**

- **Standards first** — built on OpenTelemetry standards by default
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

  HTTP:4000 ──┐     ┌─ Processor ─┐        ┌───────────┐        ┌─ HTTP API
              ├──→  │  + parser   │──→     │  DuckDB   │──→     │  (agents, scripts)
  stdin     ──┘     │             │        │           │        └──────────────
     ↑              └─────────────┘        │           │        ┌─ Socket RPC
     │                    ↓                │           │──→     │  (lotus-tui)
  SourceMux          InsertBuffer          │           │        └──────────────
                    (batch append)         └───────────┘
```

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

## License

See [LICENSE](LICENSE) for details.
