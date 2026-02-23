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
  <a href="#themes">Themes</a>
</p>

<p align="center">
  <img src="assets/Screenshot 2026-02-23 at 10.26.18.png" alt="Lotus running" width="500" />
</p>

<p align="center">
  <img src="assets/Screenshot 2026-02-23 at 10.25.45.png" alt="Lotus TUI dashboard" width="700" />
</p>

> **Early alpha** &mdash; not ready for production use.

`[ ] Logs` &ensp; `[ ] Metrics` &ensp; `[ ] Product Analytics`

## What is Lotus?

Lotus is a unified observability tool built for AI agents from first principles. It's a headless service that takes in data from TCP streams or stdin, stores everything in DuckDB, and gives you a **read-only HTTP API** to work with. Agents query live and retained data directly via SQL, no SDKs, no client libraries, no abstractions getting in the way.

There's also a **TUI client** (`lotus-cli`) that connects to the running service over a Unix socket for when a human wants a terminal dashboard. It's a standalone CLI that plugs into the socket RPC layer, completely separate from the service itself.

**Philosophy:**

**Simplicity** &mdash; no abstractions, no indirections, no layers that don't need to exist

**Imperative** &mdash; drop the binary on a machine, run it, done. Zero configuration needed

**AI agent first** &mdash; the HTTP API is the primary read surface, designed for programmatic access

**Easily extendable** &mdash; thin application layer makes it straightforward to add new inputs or read surfaces

**DuckDB is the single source of truth** &mdash; all state derives from SQL queries, no in-memory caches or secondary stores

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

## Use Cases

**Run a command and watch logs in the TUI**

Pipe any process into Lotus and get a live terminal dashboard. Useful for long-running scripts, build pipelines, or agent loops where you want structured, searchable logs without leaving the terminal.

```bash
my-agent run | lotus
# then in another terminal:
lotus-cli
```

> Tailscale setup recommended — Lotus runs locally and is not exposed to the internet, so Tailscale gives you secure remote access to the TUI and HTTP API from any device on your tailnet.

**Connect an autonomous agent to analyze your data**

Connect your autonomous agent (e.g. OpenClaw) to the Lotus HTTP API. A thin surface layer sits between agents and the data — flexible enough to support any query pattern while acting as a secure boundary. Agents read your logs and analytics, then act on them: opening PRs to fix recurring errors, making business decisions from product analytics, auto-scaling, incident triage, or any workflow you wire up.

**Observability backend for multi-agent systems**

When you have multiple agents running concurrently, pipe all their output into Lotus over TCP. Query across all agents from a single DuckDB instance — correlate events, track which agent did what, spot failures across the fleet.

**Local development dashboard**

Use Lotus as a lightweight alternative to spinning up Grafana/Loki/Prometheus during development. One binary, zero config, instant structured logs with a TUI.

**CI/CD pipeline debugging**

Pipe CI job output into Lotus, then query specific error patterns or timing data after the run. Faster than scrolling through raw build logs.

## Themes

Lotus ships with 12 color themes. Set the theme in your config file:

```yaml
skin: dracula
```

Available skins: `lotus-dark`, `lotus-light`, `dracula`, `github-light`, `gruvbox`, `matrix`, `monokai`, `nord`, `solarized-dark`, `solarized-light`, `spring`, `vs-code-light`.

## License

See [LICENSE](LICENSE) for details.
