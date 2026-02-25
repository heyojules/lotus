package backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultInterval = 6 * time.Hour
	defaultKeepLast = 24
)

// Manager runs periodic local snapshots and optional remote uploads.
type Manager struct {
	store    Snapshotter
	cfg      Config
	uploader Uploader

	done chan struct{}
	wg   sync.WaitGroup
}

// NewManager initializes backup manager. It returns nil when backups are disabled.
func NewManager(store Snapshotter, cfg Config) (*Manager, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if store == nil {
		return nil, fmt.Errorf("backup: nil snapshotter")
	}
	if strings.TrimSpace(store.DBPath()) == "" {
		return nil, fmt.Errorf("backup: db-path is empty (in-memory store)")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	if strings.TrimSpace(cfg.LocalDir) == "" {
		return nil, fmt.Errorf("backup: local-dir is required when backup is enabled")
	}
	if cfg.KeepLast <= 0 {
		cfg.KeepLast = defaultKeepLast
	}
	if err := os.MkdirAll(cfg.LocalDir, 0755); err != nil {
		return nil, fmt.Errorf("backup: create local-dir: %w", err)
	}

	var uploader Uploader
	if strings.TrimSpace(cfg.BucketURL) != "" {
		s3u, err := NewS3Uploader(S3Config{
			BucketURL:    cfg.BucketURL,
			Endpoint:     cfg.S3Endpoint,
			Region:       cfg.S3Region,
			AccessKey:    cfg.S3AccessKey,
			SecretKey:    cfg.S3SecretKey,
			SessionToken: cfg.S3SessionToken,
			UseSSL:       cfg.S3UseSSL,
			ContentType:  "application/octet-stream",
		})
		if err != nil {
			return nil, fmt.Errorf("backup: init s3 uploader: %w", err)
		}
		uploader = s3u
	}

	m := &Manager{
		store:    store,
		cfg:      cfg,
		uploader: uploader,
		done:     make(chan struct{}),
	}

	// Startup snapshot to reduce recovery point after restarts.
	if err := m.RunOnce(context.Background()); err != nil {
		log.Printf("backup: startup snapshot failed: %v", err)
	}

	m.wg.Add(1)
	go m.loop()
	return m, nil
}

func (m *Manager) loop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.RunOnce(context.Background()); err != nil {
				log.Printf("backup: periodic snapshot failed: %v", err)
			}
		case <-m.done:
			return
		}
	}
}

// RunOnce creates one local snapshot, uploads it when configured, and prunes old local copies.
func (m *Manager) RunOnce(ctx context.Context) error {
	fileName := fmt.Sprintf("lotus-%s.duckdb", time.Now().UTC().Format("20060102-150405"))
	localPath := filepath.Join(m.cfg.LocalDir, fileName)

	if err := m.store.SnapshotTo(localPath); err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	log.Printf("backup: created snapshot %s", localPath)

	if m.uploader != nil {
		if err := m.uploader.UploadFile(ctx, localPath); err != nil {
			return fmt.Errorf("upload: %w", err)
		}
		log.Printf("backup: uploaded snapshot %s", filepath.Base(localPath))
	}

	if err := pruneLocalBackups(m.cfg.LocalDir, m.cfg.KeepLast); err != nil {
		return fmt.Errorf("prune local backups: %w", err)
	}
	return nil
}

// Stop terminates the periodic backup loop.
func (m *Manager) Stop() {
	close(m.done)
	m.wg.Wait()
}

func pruneLocalBackups(localDir string, keepLast int) error {
	if keepLast <= 0 {
		return nil
	}

	matches, err := filepath.Glob(filepath.Join(localDir, "lotus-*.duckdb"))
	if err != nil {
		return err
	}
	if len(matches) <= keepLast {
		return nil
	}

	sort.Slice(matches, func(i, j int) bool {
		// timestamp is embedded in filename and lexical sort matches chronology
		return matches[i] > matches[j]
	})

	for _, oldPath := range matches[keepLast:] {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
