package duckdb

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrInMemoryStore indicates the store uses an in-memory DB and cannot be snapshotted.
var ErrInMemoryStore = errors.New("duckdb: in-memory store cannot be snapshotted")

// DBPath returns the configured DuckDB path. Empty means in-memory DB.
func (s *Store) DBPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dbPath
}

// SnapshotTo flushes and copies the on-disk DuckDB database file to dstPath.
// It serializes CHECKPOINT under the store write lock, then copies the DB file
// outside the lock to avoid stalling reads/writes for large files.
func (s *Store) SnapshotTo(dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("create snapshot dir: %w", err)
	}

	// Serialize checkpoint with DB operations for a clean snapshot boundary.
	s.mu.Lock()
	dbPath := s.dbPath
	if dbPath == "" {
		s.mu.Unlock()
		return ErrInMemoryStore
	}
	if _, err := s.db.Exec("CHECKPOINT"); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("checkpoint: %w", err)
	}
	s.mu.Unlock()

	// Copy outside the store lock so ingestion and queries are minimally blocked.
	if err := copyFile(dbPath, dstPath); err != nil {
		return fmt.Errorf("copy duckdb file: %w", err)
	}
	return nil
}

func copyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	tmp := dstPath + ".tmp"
	dst, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := dst.Sync(); err != nil {
		dst.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dstPath)
}
