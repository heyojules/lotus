package journal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tinytelemetry/lotus/internal/model"
)

func TestAppendReplayCommit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ingest.journal")

	j, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = j.Close() })

	rec1 := &model.LogRecord{
		Timestamp: time.Now().UTC(),
		Level:     "INFO",
		Message:   "first",
		App:       "default",
		Source:    "tcp",
	}
	rec2 := &model.LogRecord{
		Timestamp: time.Now().UTC(),
		Level:     "ERROR",
		Message:   "second",
		App:       "api",
		Source:    "tcp",
	}

	seq1, err := j.Append(rec1)
	if err != nil {
		t.Fatalf("Append rec1: %v", err)
	}
	seq2, err := j.Append(rec2)
	if err != nil {
		t.Fatalf("Append rec2: %v", err)
	}
	if seq2 <= seq1 {
		t.Fatalf("sequence did not advance: seq1=%d seq2=%d", seq1, seq2)
	}

	if err := j.Commit(seq1); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var replayed []string
	err = j.Replay(func(_ uint64, r *model.LogRecord) error {
		replayed = append(replayed, r.Message)
		return nil
	})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if len(replayed) != 1 || replayed[0] != "second" {
		t.Fatalf("Replay messages=%v, want [second]", replayed)
	}
}

func TestOpenIgnoresPartialTrailingLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ingest.journal")

	j, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_, err = j.Append(&model.LogRecord{
		Timestamp: time.Now().UTC(),
		Level:     "INFO",
		Message:   "ok",
		App:       "default",
		Source:    "tcp",
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := j.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Simulate torn write.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	if _, err := f.WriteString(`{"seq":999,"record":`); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close torn writer: %v", err)
	}

	j2, err := Open(path)
	if err != nil {
		t.Fatalf("Open second: %v", err)
	}
	defer func() { _ = j2.Close() }()

	var replayed []string
	err = j2.Replay(func(_ uint64, r *model.LogRecord) error {
		replayed = append(replayed, r.Message)
		return nil
	})
	if err != nil {
		t.Fatalf("Replay second: %v", err)
	}
	if len(replayed) != 1 || replayed[0] != "ok" {
		t.Fatalf("Replay after torn write=%v, want [ok]", replayed)
	}
}
