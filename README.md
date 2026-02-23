<p align="center">
  <img src="assets/26e6c9d4-1e0a-42c9-a7b2-cef549bf96db-1.png" alt="Lotus Logo" width="280" />
</p>

<h1 align="center">Lotus</h1>
<p align="center"><strong>Pragmatic Data Warehouse for Logs</strong></p>
<p align="center">
  Real-time log ingestion, analytics, and visualization powered by DuckDB.
</p>

<p align="center">
  <a href="#features">Features</a> &bull;
  <a href="#architecture">Architecture</a> &bull;
  <a href="#getting-started">Getting Started</a> &bull;
  <a href="#configuration">Configuration</a> &bull;
  <a href="#http-api">HTTP API</a> &bull;
  <a href="#themes">Themes</a>
</p>

---

## What is Lotus?

Lotus is a long-running log ingestion and analytics service that stores everything in DuckDB and exposes two read surfaces: a rich **Terminal UI** for human operators and a **read-only HTTP API** for AI agents and external tools. It ingests logs from TCP streams or stdin, normalizes them across popular logging frameworks, and derives all analytics directly from SQL queries against the database.

There is no intermediate business logic layer. DuckDB is the single source of truth&mdash;the application layer is intentionally thin, translating queries into views and keeping everything else out of the way.

## Features

- **Real-time terminal dashboard** &mdash; severity trends, word frequency, attribute analysis, pattern clustering, and heatmaps rendered in the terminal with Bubble Tea
- **Multi-format log ingestion** &mdash; auto-detects and normalizes Pino, Bunyan, Winston, Zerolog, Zap, Logrus, and plain text
- **DuckDB-powered analytics** &mdash; all state derived from SQL queries; no in-memory caches or secondary stores
- **Read-only SQL API** &mdash; safe HTTP endpoint for AI tools and scripts to query live and retained logs
- **Multi-app isolation** &mdash; app-scoped views in a single database without separate pipelines
- **Pattern clustering** &mdash; automatic log pattern recognition using the Drain3 algorithm
- **Plugin-based input** &mdash; TCP server (NDJSON) and stdin sources with a multiplexer architecture
- **12 color themes** &mdash; Dracula, Nord, Gruvbox, Solarized, Monokai, Matrix, and more
- **Configurable retention** &mdash; automatic cleanup of logs older than N days
- **Non-blocking write path** &mdash; buffered channels and batched inserts ensure ingestion never blocks on I/O

## Architecture

```
Input Plugins          Processing              Storage              Presentation
─────────────       ──────────────          ─────────────       ─────────────────

  TCP:4000  ──┐     ┌─ Processor ─┐        ┌───────────┐        ┌─ TUI Dashboard
              ├──→  │  parse +    │──→     │  DuckDB   │──→     │ (Bubble Tea)
  stdin     ──┘     │  normalize  │        │  logs     │        └──────────────
     ↑              └─────────────┘        │  table    │        ┌─ HTTP API
     │                    ↓                │           │──→     │  (Gin)
  SourceMux          InsertBuffer          │           │        └──────────────
                    (batch append)         │           │        ┌─ Socket RPC
                                           │           │        │(CLI ↔ Service)
                                           └───────────┘        └──────────────
```

**Core principles:**

1. **DuckDB is the single source of truth** &mdash; all state derives from database queries
2. **Append-only ingestion** &mdash; logs are immutable once written
3. **Thin application layer** &mdash; minimal logic, mostly query translation
4. **Non-blocking write path** &mdash; ingestion never blocks on database I/O
5. **Plugin-based input, fixed output** &mdash; inputs are extensible; storage and read surfaces are stable

## Getting Started

### Prerequisites

- Go 1.25+
- Make

### Build

```bash
# Build both the server and CLI
make build

# Or build individually
make build-lotus    # Headless server
make build-cli      # TUI client
```

Binaries are output to `./build/`.

### Run

**Option 1: Pipe logs directly via stdin**

```bash
tail -f /var/log/app.log | ./build/lotus
```

**Option 2: Start the server and send logs over TCP**

```bash
# Start the Lotus server (TCP on port 4000, API on port 3000)
./build/lotus

# Send logs from another terminal
cat logs.json | nc localhost 4000
```

**Option 3: Connect the TUI client to a running server**

```bash
# Start the headless server
./build/lotus

# In another terminal, launch the TUI
./build/lotus-cli
```

### Quick test with sample data

```bash
cat tests/test.json | nc localhost 4000
```

## Configuration

Lotus reads configuration from `~/.config/lotus/config.yml`.

| Parameter | Default | Description |
|---|---|---|
| `tcp-enabled` | `true` | Enable TCP ingest server |
| `tcp-port` | `4000` | TCP listen port |
| `api-enabled` | `true` | Enable HTTP API |
| `api-port` | `3000` | HTTP API port |
| `db-path` | `~/.local/share/lotus/lotus.duckdb` | Database file location |
| `update-interval` | `2s` | TUI refresh rate |
| `log-buffer` | `1000` | TUI log display buffer size |
| `skin` | `default` | Color theme name |
| `query-timeout` | `30s` | DuckDB query timeout |
| `insert-batch-size` | `2000` | Records per batch insert |
| `insert-flush-interval` | `100ms` | Batch flush timeout |
| `log-retention` | `30` | Days to retain logs (0 = disabled) |
| `use-log-time` | `false` | Use original log timestamps instead of ingestion time |

**Default paths:**

| Resource | Path |
|---|---|
| Config | `~/.config/lotus/config.yml` |
| Database | `~/.local/share/lotus/lotus.duckdb` |
| Socket | `~/.local/share/lotus/lotus.sock` |
| Log file | `~/.config/lotus/lotus.log` |

## HTTP API

The HTTP API provides read-only access to log data, designed for AI tools and automation.

### Endpoints

**`GET /api/health`** &mdash; Service health and uptime

```bash
curl http://localhost:3000/api/health
```

**`GET /api/schema`** &mdash; Database schema and table statistics

```bash
curl http://localhost:3000/api/schema
```

**`POST /api/query`** &mdash; Execute read-only SQL queries (max 1000 rows)

```bash
curl -X POST http://localhost:3000/api/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT level, count(*) FROM logs GROUP BY level"}'
```

## Log Formats

Lotus automatically detects and normalizes logs from these frameworks:

| Framework | Language | Format |
|---|---|---|
| Pino | Node.js | JSON with numeric levels |
| Bunyan | Node.js | JSON |
| Winston | Node.js | JSON |
| Zerolog | Go | JSON |
| Zap | Go | JSON |
| Logrus | Go | JSON |
| Plain text | Any | Line-based |

Logs are sent as newline-delimited JSON (NDJSON) over TCP or piped via stdin.

## Data Model

All logs are stored in a single `logs` table:

```sql
id              INTEGER        -- Auto-increment primary key
timestamp       TIMESTAMP      -- Normalized ingestion time
orig_timestamp  TIMESTAMP      -- Original timestamp from source
level           VARCHAR        -- TRACE / DEBUG / INFO / WARN / ERROR / FATAL
level_num       INTEGER        -- Numeric severity (10-60)
message         VARCHAR        -- Log message body
raw_line        VARCHAR        -- Original unparsed line
service         VARCHAR        -- Service name from attributes
hostname        VARCHAR        -- Source host
pid             INTEGER        -- Process ID
attributes      JSON           -- Arbitrary key-value metadata
source          VARCHAR        -- "tcp" or "stdin"
app             VARCHAR        -- Application name (default: "default")
```

Indexed on: `timestamp`, `level`, `service`, `app`, `(app, timestamp)`.

## Themes

Lotus ships with 12 color themes. Set the theme in your config file:

```yaml
skin: dracula
```

Available skins: `controltheory-dark`, `controltheory-light`, `dracula`, `github-light`, `gruvbox`, `matrix`, `monokai`, `nord`, `solarized-dark`, `solarized-light`, `spring`, `vs-code-light`.

## Business Use Cases

1. **Incident response** &mdash; Real-time severity trends, top services/hosts monitoring, and drill-down capabilities directly in the terminal
2. **AI-assisted operations** &mdash; Read-only SQL API access so AI tools can safely inspect live and retained logs without risk of mutation
3. **Multi-app visibility** &mdash; App-scoped views in a single database without maintaining separate logging pipelines

## Project Structure

```
cmd/
  lotus/           Server: ingestion, storage, HTTP API
  lotus-cli/       TUI client (connects via Unix socket RPC)
internal/
  duckdb/          Storage layer, batch inserts, queries, retention
  ingest/          Log parsing, normalization, multi-format extraction
  tui/             Terminal UI components (Bubble Tea)
  httpserver/      REST API (Gin)
  logsource/       Input abstractions (TCP, stdin)
  socketrpc/       Unix socket IPC between server and CLI
  drain3/          Log pattern clustering
  model/           Shared types and interfaces
skins/             Color theme definitions (YAML)
docs/              Architecture and design documentation
tests/             Sample log data and generation scripts
```

## Development

```bash
make build      # Build all binaries
make test       # Run tests
make run        # Build and run the server
make run-cli    # Build and run the TUI client
make clean      # Remove build artifacts
make prune      # Delete the database file
```

## License

See [LICENSE](LICENSE) for details.
