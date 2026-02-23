<p align="center">
  <img src="assets/26e6c9d4-1e0a-42c9-a7b2-cef549bf96db-1.png" alt="Lotus Logo" width="280" />
</p>

<h1 align="center">Lotus</h1>
<p align="center"><strong>Unified observability tool built for agents from first principles</strong></p>

<p align="center">
  <code>[ ] Logs</code>&ensp;&ensp;<code>[ ] Metrics</code>&ensp;&ensp;<code>[ ] Product Analytics</code>
</p>

<p align="center">
  Fully imperative. Runs without any configuration needed.
</p>

<p align="center">
  <a href="#architecture">Architecture</a> &bull;
  <a href="#themes">Themes</a>
</p>

---

> **Early alpha** &mdash; not ready for production use.

## What is Lotus?

Lotus is a unified observability tool built for AI agents from first principles. It is a headless service that ingests data from TCP streams or stdin, stores everything in DuckDB, and exposes a **read-only HTTP API** as its primary interface. Agents query live and retained data directly via SQL&mdash;no SDKs, no client libraries, no abstractions.

A separate **TUI client** (`lotus-cli`) connects to the running service over a Unix socket for human operators who want a terminal dashboard. The TUI is not part of the service&mdash;it is a standalone CLI that plugs into the socket RPC layer.

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

## Themes

Lotus ships with 12 color themes. Set the theme in your config file:

```yaml
skin: dracula
```

Available skins: `controltheory-dark`, `controltheory-light`, `dracula`, `github-light`, `gruvbox`, `matrix`, `monokai`, `nord`, `solarized-dark`, `solarized-light`, `spring`, `vs-code-light`.

## License

See [LICENSE](LICENSE) for details.
