<p align="center">
  <img src="assets/26e6c9d4-1e0a-42c9-a7b2-cef549bf96db-1.png" alt="Lotus Logo" width="280" />
</p>

<h1 align="center">Lotus</h1>
<p align="center"><strong>Unified observability tool built for agents from first principles</strong></p>

<p align="center">
  Fully imperative. Runs without any configuration needed.
</p>

<p align="center">
  <a href="#architecture">Architecture</a> &bull;
  <a href="#reliability-and-tuning">Reliability</a> &bull;
  <a href="#themes">Themes</a>
</p>

> **Early alpha** &mdash; not ready for production use.

> *"Imperative for human, declarative for agent"*

`[ ] Logs` &ensp; `[ ] Metrics` &ensp; `[ ] Product Analytics`

## What is Lotus?

Lotus is a unified observability tool built for AI agents from first principles. It's a headless service that takes in data from TCP streams or stdin, stores everything in DuckDB, and gives you a **read-only HTTP API** to work with. Agents query live and retained data through a thin query layer, no SDKs, no client libraries, no abstractions getting in the way.

There's also a **TUI client** (`lotus-cli`) that connects to the running service over a Unix socket for when a human wants a terminal dashboard. It's a standalone CLI that plugs into the socket RPC layer, completely separate from the service itself.

**Philosophy:**

- **Simplicity** &mdash; no abstractions, no indirections, no layers that don't need to exist
- **Imperative** &mdash; drop the binary on a machine, run it, done. Zero configuration needed
- **AI agent first** &mdash; the HTTP API is the primary read surface, designed for programmatic access
- **Easily extendable** &mdash; thin application layer makes it straightforward to add new inputs or read surfaces
- **DuckDB is the single source of truth** &mdash; all state derives from SQL queries, no in-memory caches or secondary stores

## Architecture

```
Input Plugins          Processing              Storage              Read Surfaces
─────────────       ──────────────          ─────────────       ─────────────────

  TCP:4000  ──┐     ┌─ Processor ─┐        ┌───────────┐        ┌─ HTTP API
              ├──→  │  parse +    │──→     │  DuckDB   │──→     │  (agents, scripts)
  stdin     ──┘     │  normalize  │        │           │        └──────────────
     ↑              └─────────────┘        │           │        ┌─ Socket RPC
     │                    ↓                │           │──→     │  (lotus-cli TUI)
  SourceMux          InsertBuffer          │           │        └──────────────
                    (batch append)         └───────────┘
```

More detailed layer docs and interface contracts:

- `docs/layers/README.md`
- `docs/layers/interfaces.md`

## Reliability and Tuning

Lotus is designed to absorb short spikes and apply backpressure under sustained load:

- Buffered ingest path (`tcpserver` + `SourceMux` + `InsertBuffer`)
- Batched DuckDB writes via async flush worker
- Read-query concurrency gate to prevent query storms from starving writes
- Overload signaling on read surfaces (`HTTP 503`, socket RPC `-32001`) for retryable pressure

Key runtime tuning options (optional):

```yaml
mux-buffer-size: 50000
insert-batch-size: 2000
insert-flush-interval: 100ms
insert-flush-queue-size: 64
max-concurrent-queries: 8
```

Notes:

- Defaults are set in code; configuration is optional.
- Very large single log lines are capped at 1MB per line on TCP/stdin inputs.

## Themes

Lotus ships with 12 color themes. Set the theme in your config file:

```yaml
skin: dracula
```

Available skins: `lotus-dark`, `lotus-light`, `dracula`, `github-light`, `gruvbox`, `matrix`, `monokai`, `nord`, `solarized-dark`, `solarized-light`, `spring`, `vs-code-light`.

## Screenshots

<p align="center">
  <img src="assets/Screenshot 2026-02-23 at 10.26.18.png" alt="Lotus running" width="500" />
</p>

<p align="center">
  <img src="assets/Screenshot 2026-02-23 at 10.25.45.png" alt="Lotus TUI dashboard" width="700" />
</p>

## License

See [LICENSE](LICENSE) for details.
