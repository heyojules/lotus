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

> [!WARNING]
> **Still in heavy development** &mdash; not ready for production use.

> *"Imperative for human, declarative for agent"*

## What is Lotus?

Lotus is a unified observability tool built for AI agents from first principles. It's a headless service that takes in data from TCP streams or stdin, stores everything in DuckDB, and gives you a **read-only HTTP API** to work with. Agents query live and retained data through a thin query layer, no SDKs, no client libraries, no abstractions getting in the way.

There's also a **TUI client** (`lotus-tui`) that connects to the running service over a Unix socket for when a human wants a terminal dashboard. It's a standalone CLI that plugs into the socket RPC layer, completely separate from the service itself.

**Philosophy:**

- **Simplicity** &mdash; no abstractions, no indirections, no layers that don't need to exist
- **Imperative** &mdash; drop the binary on a machine, run it, done. Zero configuration needed
- **AI agent first** &mdash; the HTTP API is the primary read surface, designed for programmatic access
- **Easily extendable** &mdash; thin application layer makes it straightforward to add new inputs or read surfaces
- **DuckDB is the single source of truth** &mdash; all state derives from SQL queries, no in-memory caches or secondary stores

## This exist because

- **You shouldn't need motivation to set up persistant logging.** Drop a binary, pipe your output, done.
- **Agents should fix your bugs, not you.** Lotus is a tiny data warehouse that AI agents can query autonomously to find and fix problems while you focus on things that truly matter.
- **Instant TUI mount and search.** Need a specific log? Connect and query. No dashboards to configure, no pipelines to deploy, no complexity standing between you and your data.


**Observation categories**

- [x] Logs
- [ ] Metrics
- [ ] Analytics

## Architecture

```
Input Plugins          Processing              Storage              Read Surfaces
─────────────       ──────────────          ─────────────       ─────────────────

  TCP:4000  ──┐     ┌─ Processor ─┐        ┌───────────┐        ┌─ HTTP API
              ├──→  │  parse +    │──→     │  DuckDB   │──→     │  (agents, scripts)
  stdin     ──┘     │  normalize  │        │           │        └──────────────
     ↑              └─────────────┘        │           │        ┌─ Socket RPC
     │                    ↓                │           │──→     │  (lotus-tui)
  SourceMux          InsertBuffer          │           │        └──────────────
                    (batch append)         └───────────┘
```

More detailed layer docs and interface contracts:

- `docs/layers/README.md`
- `docs/layers/interfaces.md`
- `docs/operations/rsyslog-forwarder.md`

## Ingest Topologies

Lotus supports two practical ingest modes:

1. Same machine via `stdin` pipe.
2. Other machines via TCP on port `4000`.

### Same machine (pipe to stdin)

When stdin is piped, the `stdin` input plugin is enabled automatically:

```sh
your-app 2>&1 | lotus
```

### Other machine (TCP :4000)

TCP ingest is enabled by default on port `4000`, but Lotus binds to localhost unless configured otherwise.

Default runtime bind:

```yaml
host: 127.0.0.1
tcp-port: 4000
# resolves to tcp-addr: 127.0.0.1:4000 when tcp-addr is not set
```

To accept logs from another machine, either set `host` to a reachable interface, or set `tcp-addr` directly:

```yaml
host: 0.0.0.0
# (optional) tcp-port: 4000
```

```yaml
tcp-addr: 0.0.0.0:4000
# or a specific NIC/IP, e.g. 10.0.0.12:4000
```

Then send newline-delimited logs/NDJSON to `<lotus-host>:4000`.

> [!NOTE]
> TCP ingest currently has no built-in TLS/auth. Expose it only on trusted networks or behind a secure tunnel/proxy.

## Production Durable Forwarding (Recommended)

For production, use `rsyslog` as a local durable forwarder and avoid direct `app | lotus` pipelines.

Why:

- Pipe chains are not durable across Lotus restarts/redeploys.
- `rsyslog` supports disk-backed action queues and retry-until-recovered delivery.
- You keep a simple local topology (`journald -> rsyslog -> lotus tcp:4000`) without introducing brokers.

Quick start:

1. Copy [`configs/rsyslog/lotus-local-forwarder.conf`](configs/rsyslog/lotus-local-forwarder.conf) to `/etc/rsyslog.d/20-lotus-forwarder.conf`.
2. Ensure Lotus listens on localhost TCP `:4000`.
3. Restart `rsyslog`.
4. Validate ingress with `logger -t lotus-test "hello from rsyslog"` and confirm it appears in Lotus.

Full setup, failure drill, and monitoring checklist:

- [`docs/operations/rsyslog-forwarder.md`](docs/operations/rsyslog-forwarder.md)

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

## Processor Modes

Input plugins and processors are decoupled, so processor strategy is swappable without changing ingest transport wiring.

```yaml
processor: parse
```

Supported modes:

- `parse` (default): JSON/text parsing + normalization into canonical records.
- `passthrough`: lightweight fallback-record path with minimal processing for high-throughput cases.

## Themes

Lotus ships with 12 color themes. Set the theme in your config file:

```yaml
skin: dracula
```

Available skins: `lotus-dark`, `lotus-light`, `dracula`, `github-light`, `gruvbox`, `matrix`, `monokai`, `nord`, `solarized-dark`, `solarized-light`, `spring`, `vs-code-light`.

## Screenshots

<p align="center">
  <img src="assets/Screenshot 2026-02-23 at 22.22.09.png" alt="Lotus running" width="500" />
</p>

<p align="center">
  <img src="assets/Screenshot 2026-02-23 at 10.25.45.png" alt="Lotus TUI dashboard" width="700" />
</p>

## License

See [LICENSE](LICENSE) for details.
