package duckdb

import (
	"sync"
	"testing"
	"time"
)

func TestInsertBuffer_AddAndStop(t *testing.T) {
	store := newTestStore(t)
	buf := NewInsertBuffer(store)

	// Add records
	for i := 0; i < 10; i++ {
		buf.Add(&LogRecord{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "test message",
			Source:    "stdin",
			App:       "default",
		})
	}

	// Stop should flush all pending records
	buf.Stop()

	count, err := store.TotalLogCount(QueryOpts{})
	if err != nil {
		t.Fatalf("TotalLogCount: %v", err)
	}
	if count != 10 {
		t.Errorf("after Stop, TotalLogCount = %d, want 10", count)
	}
}

func TestInsertBuffer_BatchThreshold(t *testing.T) {
	store := newTestStore(t)
	buf := NewInsertBuffer(store)

	// Add more than maxBatch (2000) records to trigger immediate flush
	for i := 0; i < 2100; i++ {
		buf.Add(&LogRecord{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "batch test",
			Source:    "stdin",
			App:       "default",
		})
	}

	buf.Stop()

	count, err := store.TotalLogCount(QueryOpts{})
	if err != nil {
		t.Fatalf("TotalLogCount: %v", err)
	}
	if count != 2100 {
		t.Errorf("after batch insert, TotalLogCount = %d, want 2100", count)
	}
}

func TestInsertBuffer_ConcurrentAdd(t *testing.T) {
	store := newTestStore(t)
	buf := NewInsertBuffer(store)

	var wg sync.WaitGroup
	numGoroutines := 10
	recordsPerGoroutine := 50

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < recordsPerGoroutine; i++ {
				buf.Add(&LogRecord{
					Timestamp: time.Now(),
					Level:     "INFO",
					Message:   "concurrent test",
					Source:    "stdin",
					App:       "default",
				})
			}
		}()
	}

	wg.Wait()
	buf.Stop()

	expected := int64(numGoroutines * recordsPerGoroutine)
	count, err := store.TotalLogCount(QueryOpts{})
	if err != nil {
		t.Fatalf("TotalLogCount: %v", err)
	}
	if count != expected {
		t.Errorf("concurrent insert TotalLogCount = %d, want %d", count, expected)
	}
}

func TestInsertBuffer_StopIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	buf := NewInsertBuffer(store)

	buf.Add(&LogRecord{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "idempotent stop",
		Source:    "stdin",
		App:       "default",
	})

	buf.Stop()
	buf.Stop()

	count, err := store.TotalLogCount(QueryOpts{})
	if err != nil {
		t.Fatalf("TotalLogCount: %v", err)
	}
	if count != 1 {
		t.Errorf("after double Stop, TotalLogCount = %d, want 1", count)
	}
}
