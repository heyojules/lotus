# Lotus Architecture

```
╔══════════════════════════════════════════════════════════════════════════════════════╗
║                          LOTUS — SOFTWARE ARCHITECTURE                             ║
╚══════════════════════════════════════════════════════════════════════════════════════╝


 ┌─────────────────────────────────────────────────────────────────────────────────┐
 │  EXTERNAL CONSUMERS                                                            │
 │                                                                                │
 │   Terminal (TTY)              AI Agents / curl / SDKs                           │
 │   ┌──────────────┐           ┌──────────────────────┐                          │
 │   │  Bubble Tea  │           │   HTTP Client         │                          │
 │   │  TUI         │           │   :3000               │                          │
 │   └──────┬───────┘           └──────────┬───────────┘                          │
 │          │                              │                                      │
 └──────────┼──────────────────────────────┼──────────────────────────────────────┘
            │                              │
            │ LogQuerier                   │ QueryStore
            │ (27 methods)                 │ (SchemaQuerier + TotalLogCount)
            │                              │
┏━━━━━━━━━━━┷━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┷━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                                                                                   ┃
┃  PRESENTATION LAYER                                                               ┃
┃                                                                                   ┃
┃  ┌─────────────────────────────────┐    ┌──────────────────────────────────┐      ┃
┃  │ tui.DashboardModel             │    │ httpserver.Server  :3000          │      ┃
┃  │ internal/tui/model.go          │    │ internal/httpserver/server.go     │      ┃
┃  │                                │    │                                  │      ┃
┃  │ State:                         │    │ GET  /api/health                 │      ┃
┃  │   FilterState (regex, search)  │    │   → status, uptime, log_count   │      ┃
┃  │   SidebarState (app selection) │    │                                  │      ┃
┃  │   ModalState (7 modal types)   │    │ GET  /api/schema                │      ┃
┃  │   NavigationState (sections)   │    │   → description, tables, counts │      ┃
┃  │   LogViewState (entries/scroll)│    │                                  │      ┃
┃  │                                │    │ POST /api/query                  │      ┃
┃  │ Charts (ChartPanel interface): │    │   → read-only SQL execution     │      ┃
┃  │  ┌───────┐ ┌──────┐ ┌──────┐  │    │     max 1000 rows               │      ┃
┃  │  │ Words │ │Attrs │ │Ptrns │  │    │     DDL/DML rejected            │      ┃
┃  │  └───────┘ └──────┘ └──────┘  │    │                                  │      ┃
┃  │  ┌────────────┐ ┌──────────┐  │    │ Middleware:                      │      ┃
┃  │  │Distribution│ │  Counts  │  │    │   recovery (panic → 500)         │      ┃
┃  │  │            │ │ (heatmap)│  │    │   timeouts (R:30s W:60s H:10s)   │      ┃
┃  │  └────────────┘ └──────────┘  │    └──────────────────────────────────┘      ┃
┃  │                                │                                              ┃
┃  │ Drain3Manager (in-memory):     │                                              ┃
┃  │   pattern clustering per-sev   │                                              ┃
┃  │   depth=4, similarity=0.5      │                                              ┃
┃  │                                │                                              ┃
┃  │ TickMsg every 2s:              │                                              ┃
┃  │   polls DuckDB via LogQuerier  │                                              ┃
┃  └─────────────────────────────────┘                                              ┃
┃                                                                                   ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
            │                              │
            │ LogQuerier                   │ SchemaQuerier
            │ interface                    │ interface
            ▼                              ▼
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                                                                                   ┃
┃  STORAGE LAYER                                                                    ┃
┃                                                                                   ┃
┃  ┌──────────────────────────────────────────────────────────────────────────┐      ┃
┃  │ duckdb.Store                            internal/duckdb/store.go        │      ┃
┃  │                                                                        │      ┃
┃  │ sync.RWMutex │ *sql.DB (embedded DuckDB) │ QueryTimeout: 30s           │      ┃
┃  │                                                                        │      ┃
┃  │ ┌──────────────────────────────────────────────────────────────┐        │      ┃
┃  │ │ logs TABLE                                                   │        │      ┃
┃  │ │                                                              │        │      ┃
┃  │ │  id              INTEGER (auto)                              │        │      ┃
┃  │ │  timestamp        TIMESTAMP                                  │        │      ┃
┃  │ │  orig_timestamp   TIMESTAMP                                  │        │      ┃
┃  │ │  level            VARCHAR        (TRACE→FATAL)               │        │      ┃
┃  │ │  level_num        INTEGER                                    │        │      ┃
┃  │ │  message          VARCHAR                                    │        │      ┃
┃  │ │  raw_line         VARCHAR                                    │        │      ┃
┃  │ │  service          VARCHAR                                    │        │      ┃
┃  │ │  hostname         VARCHAR                                    │        │      ┃
┃  │ │  pid              INTEGER                                    │        │      ┃
┃  │ │  attributes       JSON                                       │        │      ┃
┃  │ │  source           VARCHAR                                    │        │      ┃
┃  │ │  app              VARCHAR        (default 'default')         │        │      ┃
┃  │ │                                                              │        │      ┃
┃  │ │  IDX: timestamp, level, service, app, (app+timestamp)       │        │      ┃
┃  │ └──────────────────────────────────────────────────────────────┘        │      ┃
┃  │                                                                        │      ┃
┃  │ Migrations: 001_schema → 002_indexes → 003_metrics → 004_app_column   │      ┃
┃  └──────────────────────────────────────────────────────────────────────────┘      ┃
┃           ▲                                                                       ┃
┃           │                                                                       ┃
┃  ┌────────┴─────────────────────────────────────────────────────────┐              ┃
┃  │ duckdb.InsertBuffer                internal/duckdb/insert.go    │              ┃
┃  │                                                                  │              ┃
┃  │ Add(record) ─── NEVER BLOCKS ──→ pending[] (mutex-guarded)      │              ┃
┃  │                                       │                          │              ┃
┃  │                            ┌──────────┼──────────┐               │              ┃
┃  │                            ▼          ▼          ▼               │              ┃
┃  │                       size≥2000   tick 100ms   chan full         │              ┃
┃  │                            │          │       (safety valve)     │              ┃
┃  │                            ▼          ▼          │               │              ┃
┃  │                       flushChan (cap 64) ◄───────┘               │              ┃
┃  │                            │            ← DECOUPLING POINT 3 →   │              ┃
┃  │                            ▼                                     │              ┃
┃  │                       flushWorker goroutine                      │              ┃
┃  │                            │                                     │              ┃
┃  │                            ▼                                     │              ┃
┃  │                       Store.InsertLogBatch()                     │              ┃
┃  │                       (BEGIN TX → INSERT → COMMIT)               │              ┃
┃  └──────────────────────────────────────────────────────────────────┘              ┃
┃                                                                                   ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
            ▲
            │ InsertBuffer.Add(LogRecord)
            │
┏━━━━━━━━━━━┷━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                                                                                   ┃
┃  PROCESSING LAYER                                                                 ┃
┃                                                                                   ┃
┃  ┌──────────────────────────────────────────────────────────────────────────┐      ┃
┃  │ ingest.Processor                      internal/ingest/processor.go      │      ┃
┃  │                                                                        │      ┃
┃  │ ProcessLine(line string) *ProcessResult                                │      ┃
┃  │      │                                                                 │      ┃
┃  │      ├──→ tryAccumulateJSON()    multi-line { } depth tracking         │      ┃
┃  │      │                                                                 │      ┃
┃  │      ▼                                                                 │      ┃
┃  │ ┌────────────────────────────────────────────────────────┐              │      ┃
┃  │ │ ingest.Extractor               internal/ingest/        │              │      ┃
┃  │ │                                extractor.go            │              │      ┃
┃  │ │  ParseJSONLogEntry(line)                               │              │      ┃
┃  │ │    formats: pino, bunyan, winston, zerolog, zap,       │              │      ┃
┃  │ │             logrus                                     │              │      ┃
┃  │ │                                                        │              │      ┃
┃  │ │  CreateFallbackLogEntry(line)   plain text fallback    │              │      ┃
┃  │ │                                                        │              │      ┃
┃  │ │  ExtractService()   severity normalization             │              │      ┃
┃  │ │  timestamp.Parse()  multi-format detection             │              │      ┃
┃  │ └────────────────────────────────────────────────────────┘              │      ┃
┃  │      │                                                                 │      ┃
┃  │      ▼                                                                 │      ┃
┃  │ ┌─────────────────────────────┐                                        │      ┃
┃  │ │ ingest.LogEntry             │  ← DATA TRANSFORMATION                 │      ┃
┃  │ │   Timestamp     time.Time   │    raw string → structured entry       │      ┃
┃  │ │   Severity      string      │                                        │      ┃
┃  │ │   Message       string      │                                        │      ┃
┃  │ │   Attributes    map[s]s     │                                        │      ┃
┃  │ │   App           string      │                                        │      ┃
┃  │ └──────────┬──────────────────┘                                        │      ┃
┃  │            │                                                           │      ┃
┃  │     ┌──────┴──────┐                                                    │      ┃
┃  │     ▼             ▼                                                    │      ┃
┃  │  LogRecord    ProcessResult                                            │      ┃
┃  │  (→ Store)    (→ TUI update)                                           │      ┃
┃  └──────────────────────────────────────────────────────────────────────────┘      ┃
┃                                                                                   ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
            ▲
            │ <-chan string (merged stream)
            │ ← DECOUPLING POINT 2 (Bubble Tea msg loop batches up to 500 lines)
            │
┏━━━━━━━━━━━┷━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                                                                                   ┃
┃  INPUT LAYER                                                                      ┃
┃                                                                                   ┃
┃  ┌──────────────────────────────────────────────────────────────────────────┐      ┃
┃  │ SourceMultiplexer                     cmd/lotus/source_mux.go           │      ┃
┃  │                                                                        │      ┃
┃  │ []LogSource ──── fan-in (per-source goroutine) ───→ chan string (50K)  │      ┃
┃  │                                                                        │      ┃
┃  │ ← DECOUPLING POINT 1 (50,000-capacity buffered channel) →             │      ┃
┃  └──────────────────────────────────────────────────────────────────────────┘      ┃
┃           ▲                          ▲                                            ┃
┃           │                          │                                            ┃
┃    ·······│··························│·······································     ┃
┃    : LogSource interface             │                                      :     ┃
┃    :   Lines() <-chan string    InputSourcePlugin interface                 :     ┃
┃    :   Stop()                    Name() / Enabled() / Build(ctx)           :     ┃
┃    :   Name() string                                                       :     ┃
┃    :····································································:     ┃
┃           │                          │                                            ┃
┃  ┌────────┴────────┐       ┌─────────┴────────┐                                  ┃
┃  │ TCP Server      │       │ Stdin Source      │       ... future sources         ┃
┃  │ :4000           │       │ (piped input)     │                                  ┃
┃  │                 │       │                   │                                  ┃
┃  │ NDJSON/Pino     │       │ any text format   │                                  ┃
┃  │ 1MB max line    │       │                   │                                  ┃
┃  │ 100K chan buffer│       │                   │                                  ┃
┃  └─────────────────┘       └───────────────────┘                                  ┃
┃                                                                                   ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```


## Interface Boundary Map

```
  ┌──────────────────────────────────────────────────────────────────┐
  │                      INTERFACE CONTRACTS                         │
  │                                                                  │
  │  INPUT BOUNDARY                                                  │
  │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄   │
  │  LogSource                    InputSourcePlugin                  │
  │    Lines() <-chan string        Name() string                    │
  │    Stop()                       Enabled() bool                   │
  │    Name() string                Build(ctx) (LogSource, error)    │
  │                                                                  │
  │  Implementations:             Implementations:                   │
  │    TCPSource                    tcpInputPlugin                   │
  │    StdinSource                  stdinInputPlugin                 │
  │                                                                  │
  │  STORAGE / QUERY BOUNDARY                                        │
  │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄   │
  │  LogQuerier (27 methods)      SchemaQuerier (3 methods)          │
  │    TotalLogCount()              ExecuteQuery(sql)                │
  │    TopWords(limit)              GetSchemaDescription()           │
  │    TopAttributes(limit)         TableRowCounts()                 │
  │    SeverityCounts()                                              │
  │    SeverityCountsByMinute()   QueryStore (composite)             │
  │    RecentLogsFiltered(...)      = SchemaQuerier                  │
  │    ListApps()                   + TotalLogCount()                │
  │    + 10 ByApp() variants                                         │
  │                                                                  │
  │  Consumer: TUI Dashboard      Consumer: HTTP Server              │
  │  Implementation: duckdb.Store  Implementation: duckdb.Store      │
  │                                                                  │
  │  TUI COMPONENT BOUNDARY                                          │
  │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄   │
  │  ChartPanel                                                      │
  │    ID() / Title() / Render(w,h,active,sel) / ItemCount()         │
  │                                                                  │
  │  Panels: WordsPanel, AttributesPanel, PatternsPanel, CountsPanel │
  │                                                                  │
  │  FRAMEWORK BOUNDARY                                              │
  │  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄   │
  │  tea.Model (Bubble Tea)                                          │
  │    Init() / Update(msg) / View()                                 │
  │                                                                  │
  │  Implementations: simpleTuiModel (outer), DashboardModel (inner) │
  └──────────────────────────────────────────────────────────────────┘
```


## Write Path (Ingestion Flow)

Fully decoupled, non-blocking from source to storage.

```
  :4000 TCP           stdin
  NDJSON/Pino         piped text
       │                  │
       ▼                  ▼
  TCPSource          StdinSource
  .Lines()           .Lines()          ← LogSource interface
       │                  │
       └────────┬─────────┘
                │  fan-in goroutines
                ▼
       SourceMultiplexer
       chan string (cap 50,000)         ← DECOUPLING POINT 1
                │                         buffered channel absorbs bursts
                │
                ▼
       readInputBatch()
       drains up to 500 lines           ← DECOUPLING POINT 2
       returns logBatchMsg                Bubble Tea command loop
                │
                ▼
       ┌─── processLogLine() ───┐
       │                        │
       ▼                        ▼
  Processor.ProcessLine()   TUI receives
       │                    ProcessResult
       ▼                    for live display
  Extractor
  ParseJSONLogEntry()
  or CreateFallbackLogEntry()
       │
       ▼
  ingest.LogEntry ──→ duckdb.LogRecord
       │
       ▼
  InsertBuffer.Add()                    ← NEVER BLOCKS
       │
       ▼
  pending[] (mutex)
       │
  ┌────┴────┐
  ▼         ▼
 size≥2000  tick 100ms
  │         │
  ▼         ▼
  flushChan (cap 64)                    ← DECOUPLING POINT 3
       │                                  async batch queue
       ▼
  flushWorker goroutine
       │
       ▼
  Store.InsertLogBatch()
  BEGIN → INSERT batch → COMMIT
       │
       ▼
  ┌──────────┐
  │  DuckDB  │
  │  (OLAP)  │
  └──────────┘
```


## Read Path (Query Flow)

```
  ┌─────────────────────────────────────────────────────┐
  │  TUI READ PATH                                      │
  │                                                     │
  │  tea.Tick(2s) → TickMsg                             │
  │       │                                             │
  │       ▼                                             │
  │  DashboardModel.Update(TickMsg)                     │
  │       │                                             │
  │       ├─→ store.TopWords(20)                        │
  │       ├─→ store.TopAttributes(20)                   │
  │       ├─→ store.SeverityCounts()                    │
  │       ├─→ store.SeverityCountsByMinute()            │
  │       ├─→ store.RecentLogsFiltered()                │
  │       └─→ store.ListApps()                          │
  │              (all via LogQuerier interface)          │
  │       │                                             │
  │       ▼  RWMutex.RLock                              │
  │    DuckDB ──→ SELECT queries                        │
  │       │                                             │
  │       ▼                                             │
  │  Render: charts (ntcharts) + log table + modals     │
  └─────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────┐
  │  API READ PATH (for AI agents)                      │
  │                                                     │
  │  HTTP Client (agent/curl/SDK)                       │
  │       │                                             │
  │       ▼                                             │
  │  GET  /api/health  → TotalLogCount()                │
  │  GET  /api/schema  → GetSchemaDescription()         │
  │  POST /api/query   → ExecuteQuery(sql)              │
  │         (via QueryStore = SchemaQuerier + count)     │
  │       │                                             │
  │       ▼  RWMutex.RLock                              │
  │    DuckDB ──→ read-only SQL                         │
  │       │                                             │
  │       ▼                                             │
  │  JSON response → { columns, rows, row_count }       │
  └─────────────────────────────────────────────────────┘
```


## Runtime Goroutine Map

```
  Main goroutine
    └─→ tea.Program.Run() (Bubble Tea event loop)

  Background goroutines:
    ├─→ TCPServer.Accept()          (1 goroutine)
    │     └─→ per-connection handler (N goroutines)
    ├─→ StdinSource reader          (1 goroutine)
    ├─→ SourceMux.forward() × N    (1 per source)
    ├─→ InsertBuffer.tickLoop()     (1 goroutine, 100ms timer)
    ├─→ InsertBuffer.flushWorker()  (1 goroutine, batch writer)
    ├─→ HTTP Server                 (1 listener + M handler goroutines)
    └─→ VersionChecker              (1 goroutine, background check)
```


## Component Wiring (app.go)

```
  runApp(cfg)
    │
    ├──→ duckdb.NewStore(dbPath, timeout)
    │      └──→ migrate.NewRunner(db).Run()     (4 SQL migrations)
    │
    ├──→ duckdb.NewInsertBuffer(store, batchCfg)
    │      ├──→ starts flushWorker goroutine
    │      └──→ starts tickLoop goroutine
    │
    ├──→ httpserver.NewServer(":3000", store)
    │      └──→ .Start()                        (HTTP goroutine)
    │
    ├──→ tui.NewDashboardModel(store, cfg)
    │      ├──→ 4 ChartPanels created
    │      └──→ Drain3 managers initialized
    │
    ├──→ ingest.NewProcessor(insertBuffer)
    │
    └──→ tea.NewProgram(simpleTuiModel)
           └──→ .Run()
                  └──→ Init(): build sources → SourceMux → readInputBatch()
```


## Key Architectural Properties

1. **Non-blocking write path** -- ingestion never blocks on DuckDB I/O; InsertBuffer uses async channel queue with inline-flush safety valve
2. **3 decoupling points** -- (1) 50K buffered channel in SourceMux, (2) Bubble Tea message loop batching, (3) InsertBuffer's 64-slot flush channel
3. **Interface-driven boundaries** -- 5 interfaces (LogSource, InputSourcePlugin, LogQuerier, SchemaQuerier, ChartPanel) enable independent testing and extensibility
4. **Dual presentation** -- same DuckDB store serves TUI (LogQuerier, polling 2s) and HTTP API (QueryStore) without contention via RWMutex
5. **Pattern clustering in-memory** -- Drain3 runs separately from DuckDB with per-severity instances
