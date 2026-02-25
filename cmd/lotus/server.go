package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/control-theory/lotus/internal/backup"
	"github.com/control-theory/lotus/internal/duckdb"
	"github.com/control-theory/lotus/internal/httpserver"
	"github.com/control-theory/lotus/internal/ingest"
	"github.com/control-theory/lotus/internal/socketrpc"
	"golang.org/x/sync/errgroup"
)

// runServer starts headless log ingestion with the HTTP API.
func runServer(cfg appConfig) error {
	cleanupLogger := configureRuntimeLogger()
	defer cleanupLogger()

	// Initialize DuckDB store
	store, err := duckdb.NewStore(cfg.DBPath, cfg.QueryTimeout)
	if err != nil {
		return fmt.Errorf("failed to initialize DuckDB: %w", err)
	}
	defer store.Close()
	store.SetMaxConcurrentQueries(cfg.MaxConcurrentReads)

	// Create insert buffer for batched DuckDB writes
	insertBuffer := duckdb.NewInsertBuffer(store, duckdb.InsertBufferConfig{
		BatchSize:      cfg.InsertBatchSize,
		FlushInterval:  cfg.InsertFlushInterval,
		FlushQueueSize: cfg.InsertFlushQueue,
	})
	defer insertBuffer.Stop()

	// Start retention cleaner for automatic log expiry
	retentionCleaner := duckdb.NewRetentionCleaner(store, duckdb.RetentionConfig{
		RetentionDays: cfg.LogRetention,
	})
	if retentionCleaner != nil {
		defer retentionCleaner.Stop()
	}

	// Start periodic backups when enabled.
	backupManager, err := backup.NewManager(store, backup.Config{
		Enabled:        cfg.BackupEnabled,
		Interval:       cfg.BackupInterval,
		LocalDir:       cfg.BackupLocalDir,
		KeepLast:       cfg.BackupKeepLast,
		BucketURL:      cfg.BackupBucketURL,
		S3Endpoint:     cfg.BackupS3Endpoint,
		S3Region:       cfg.BackupS3Region,
		S3AccessKey:    cfg.BackupS3AccessKey,
		S3SecretKey:    cfg.BackupS3SecretKey,
		S3SessionToken: cfg.BackupS3SessionToken,
		S3UseSSL:       cfg.BackupS3UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize backups: %w", err)
	}
	if backupManager != nil {
		defer backupManager.Stop()
	}

	// Start HTTP API server if enabled
	if cfg.APIEnabled {
		apiServer := httpserver.NewServer(cfg.APIAddr, store)
		if err := apiServer.Start(); err != nil {
			return fmt.Errorf("failed to start API server: %w", err)
		}
		defer apiServer.Stop()
	}

	// Start socket RPC server for TUI IPC
	sockServer := socketrpc.NewServer(cfg.SocketPath, store)
	if err := sockServer.Start(); err != nil {
		log.Printf("Warning: failed to start socket server: %v", err)
	} else {
		defer sockServer.Stop()
	}

	// Build input plugins and source multiplexer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	plugins := buildInputPlugins(InputPluginConfig{
		TCPEnabled: cfg.TCPEnabled,
		TCPAddr:    cfg.TCPAddr,
	})

	sources := make([]NamedLogSource, 0, len(plugins))
	for _, plugin := range plugins {
		if !plugin.Enabled() {
			continue
		}
		src, err := plugin.Build(ctx)
		if err != nil {
			log.Printf("Error initializing input plugin %q: %v", plugin.Name(), err)
			continue
		}
		sources = append(sources, src)
	}

	if len(sources) == 0 {
		// Fall back to stdin if piped
		fallback := stdinInputPlugin{}
		if fallback.Enabled() {
			if src, err := fallback.Build(ctx); err == nil {
				sources = append(sources, src)
			}
		}
	}

	mux := NewSourceMultiplexer(ctx, sources, cfg.MuxBufferSize)
	mux.Start()

	// Create the configured envelope processor.
	processor, err := ingest.NewEnvelopeProcessor(cfg.Processor, insertBuffer, "")
	if err != nil {
		return fmt.Errorf("build processor: %w", err)
	}

	printStartupBanner(cfg, mux.HasSources(), processor.Name())

	// Use errgroup for concurrent goroutine lifecycle management.
	g, gctx := errgroup.WithContext(ctx)

	// Ingestion loop
	if mux.HasSources() {
		g.Go(func() error {
			for env := range mux.Lines() {
				processor.ProcessEnvelope(env)
			}
			return nil
		})
	}

	// Signal handler
	g.Go(func() error {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigCh)

		select {
		case <-sigCh:
			fmt.Println("\nShutting down...")
		case <-gctx.Done():
		}
		return nil
	})

	// Wait for either signal or all sources to close.
	if err := g.Wait(); err != nil {
		log.Printf("server: errgroup exited with error: %v", err)
	}

	cancel()
	mux.Stop()

	return nil
}

func configureRuntimeLogger() func() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	home, err := os.UserHomeDir()
	if err != nil {
		log.SetOutput(os.Stderr)
		return func() {}
	}

	logDir := filepath.Join(home, ".local", "state", "lotus")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.SetOutput(os.Stderr)
		return func() {}
	}

	logPath := filepath.Join(logDir, "lotus.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.SetOutput(os.Stderr)
		return func() {}
	}

	log.SetOutput(f)
	return func() {
		_ = f.Close()
	}
}

func printStartupBanner(cfg appConfig, _ bool, processorName string) {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	bold := lipgloss.NewStyle().Bold(true)

	check := green.Render("●")
	dot := dim.Render("●")

	logo := cyan.Bold(true).Render(`
    ╦  ╔═╗╔╦╗╦ ╦╔═╗
    ║  ║ ║ ║ ║ ║╚═╗
    ╩═╝╚═╝ ╩ ╚═╝╚═╝`)

	ver := dim.Render("v" + version)

	var lines []string
	lines = append(lines, "")
	lines = append(lines, logo)
	lines = append(lines, "    "+ver)
	lines = append(lines, "")

	separator := dim.Render("    ─────────────────────────────────")
	lines = append(lines, separator)
	lines = append(lines, "")

	// Gateway
	lines = append(lines, bold.Render("    Gateway"))
	lines = append(lines, "")

	if cfg.APIEnabled {
		lines = append(lines, fmt.Sprintf("    %s  HTTP API       %s", check, cyan.Render(cfg.APIAddr)))
	} else {
		lines = append(lines, fmt.Sprintf("    %s  HTTP API       %s", dot, dim.Render("disabled")))
	}

	if cfg.TCPEnabled {
		lines = append(lines, fmt.Sprintf("    %s  TCP Ingest     %s", check, cyan.Render(cfg.TCPAddr)))
	} else {
		lines = append(lines, fmt.Sprintf("    %s  TCP Ingest     %s", dot, dim.Render("disabled")))
	}

	lines = append(lines, fmt.Sprintf("    %s  Unix Socket    %s", check, cyan.Render(shortenPath(cfg.SocketPath))))
	lines = append(lines, "")

	// Storage
	lines = append(lines, bold.Render("    Storage"))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("    %s  Storage        %s", check, dim.Render(shortenPath(cfg.DBPath))))
	if cfg.BackupEnabled {
		lines = append(lines, fmt.Sprintf("    %s  Snapshots      %s", check, dim.Render(shortenPath(cfg.BackupLocalDir))))
	} else {
		lines = append(lines, fmt.Sprintf("    %s  Snapshots      %s", dot, dim.Render("disabled")))
	}

	lines = append(lines, "")

	// Runtime
	lines = append(lines, bold.Render("    Runtime"))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("    %s  Processor      %s", check, dim.Render(processorName)))

	lines = append(lines, "")
	lines = append(lines, bold.Render("    Config"))
	lines = append(lines, "")
	if cfg.ConfigPath != "" {
		lines = append(lines, fmt.Sprintf("    %s  Config File    %s", check, dim.Render(shortenPath(cfg.ConfigPath))))
	} else {
		lines = append(lines, fmt.Sprintf("    %s  Config File    %s", dot, dim.Render("default (no file)")))
	}

	lines = append(lines, "")
	lines = append(lines, separator)
	lines = append(lines, "")
	lines = append(lines, "    "+dim.Render("Press ")+yellow.Render("Ctrl+C")+dim.Render(" to stop"))
	lines = append(lines, "")

	fmt.Println(strings.Join(lines, "\n"))
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
