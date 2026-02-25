package duckdb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotTo_CreatesBackupFile(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "lotus.duckdb")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	err = store.InsertLogBatch([]*LogRecord{
		{
			Timestamp: time.Now(),
			Level:     "INFO",
			LevelNum:  30,
			Message:   "snapshot test",
			RawLine:   "snapshot test",
			Source:    "stdin",
			App:       "default",
		},
	})
	if err != nil {
		t.Fatalf("InsertLogBatch: %v", err)
	}

	snapshotPath := filepath.Join(t.TempDir(), "backups", "snapshot.duckdb")
	if err := store.SnapshotTo(snapshotPath); err != nil {
		t.Fatalf("SnapshotTo: %v", err)
	}

	info, err := os.Stat(snapshotPath)
	if err != nil {
		t.Fatalf("stat snapshot: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("snapshot file is empty")
	}
}

func TestSnapshotTo_InMemoryStore(t *testing.T) {
	t.Parallel()

	store, err := NewStore("")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	err = store.SnapshotTo(filepath.Join(t.TempDir(), "snapshot.duckdb"))
	if err == nil {
		t.Fatal("expected error for in-memory store")
	}
	if err != ErrInMemoryStore {
		t.Fatalf("err = %v, want %v", err, ErrInMemoryStore)
	}
}
