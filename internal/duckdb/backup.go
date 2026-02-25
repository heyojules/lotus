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
// It acquires the store write lock to serialize with reads/writes during snapshot.
func (s *Store) SnapshotTo(dstPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dbPath == "" {
		return ErrInMemoryStore
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("create snapshot dir: %w", err)
	}

	// Flush pending data and reduce WAL drift before file copy.
	if _, err := s.db.Exec("CHECKPOINT"); err != nil {
		return fmt.Errorf("checkpoint: %w", err)
	}

	if err := copyFile(s.dbPath, dstPath); err != nil {
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
