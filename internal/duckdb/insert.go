package duckdb

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// defaultFlushQueueSize is the number of batches that can be queued for async flushing.
const defaultFlushQueueSize = 64

// InsertBuffer batches log records and flushes them to DuckDB asynchronously.
// Add() never blocks on DuckDB writes - records are sent to a flush goroutine.
type InsertBuffer struct {
	store         *Store
	mu            sync.Mutex
	pending       []*LogRecord
	flushChan     chan []*LogRecord // async flush queue
	maxBatch      int
	flushInterval time.Duration
	done          chan struct{}
	wg            sync.WaitGroup
	tickWg        sync.WaitGroup // separate WaitGroup for tickLoop

	// backpressureCount tracks inline flushes for throttled logging.
	backpressureCount atomic.Int64
	lastBPLog         atomic.Int64 // unix timestamp of last backpressure log
}

// InsertBufferConfig holds tunable parameters for the insert buffer.
type InsertBufferConfig struct {
	BatchSize     int
	FlushInterval time.Duration
}

// NewInsertBuffer creates a new insert buffer that flushes to the store.
// The flush goroutine processes batches asynchronously so Add() never blocks on IO.
func NewInsertBuffer(store *Store, conf ...InsertBufferConfig) *InsertBuffer {
	batchSize := 2000
	flushInterval := 100 * time.Millisecond
	if len(conf) > 0 {
		if conf[0].BatchSize > 0 {
			batchSize = conf[0].BatchSize
		}
		if conf[0].FlushInterval > 0 {
			flushInterval = conf[0].FlushInterval
		}
	}

	b := &InsertBuffer{
		store:         store,
		pending:       make([]*LogRecord, 0, batchSize),
		flushChan:     make(chan []*LogRecord, defaultFlushQueueSize),
		maxBatch:      batchSize,
		flushInterval: flushInterval,
		done:          make(chan struct{}),
	}

	b.wg.Add(1)
	go b.flushWorker()

	b.wg.Add(1)
	b.tickWg.Add(1)
	go b.tickLoop()

	return b
}

// tickLoop periodically drains the pending buffer.
func (b *InsertBuffer) tickLoop() {
	defer b.wg.Done()
	defer b.tickWg.Done()
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.drainPending()
		case <-b.done:
			b.drainPending() // final drain
			return
		}
	}
}

// logBackpressure emits a throttled warning (at most once per 10 seconds) when
// the flush channel is full and an inline flush is triggered.
func (b *InsertBuffer) logBackpressure() {
	count := b.backpressureCount.Add(1)
	now := time.Now().Unix()
	last := b.lastBPLog.Load()
	if now-last >= 10 && b.lastBPLog.CompareAndSwap(last, now) {
		log.Printf("duckdb: backpressure — %d inline flushes (flush channel full, DuckDB falling behind)", count)
	}
}

// drainPending moves pending records to the flush channel without blocking on DuckDB.
func (b *InsertBuffer) drainPending() {
	b.mu.Lock()
	if len(b.pending) == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.pending
	b.pending = make([]*LogRecord, 0, b.maxBatch)
	b.mu.Unlock()

	// Non-blocking send to flush channel. If channel is full, flush synchronously
	// as a safety valve (this means DuckDB is falling behind).
	select {
	case b.flushChan <- batch:
	default:
		b.logBackpressure()
		if err := b.store.InsertLogBatch(batch); err != nil {
			log.Printf("duckdb flush error (inline): %v", err)
		}
	}
}

// flushWorker processes batches from the flush channel.
func (b *InsertBuffer) flushWorker() {
	defer b.wg.Done()
	for batch := range b.flushChan {
		if err := b.store.InsertLogBatch(batch); err != nil {
			log.Printf("duckdb flush error: %v", err)
		}
	}
}

// Add queues a record for batch insertion. This never blocks on DuckDB IO.
func (b *InsertBuffer) Add(record *LogRecord) {
	b.mu.Lock()
	b.pending = append(b.pending, record)
	shouldFlush := len(b.pending) >= b.maxBatch
	var batch []*LogRecord
	if shouldFlush {
		batch = b.pending
		b.pending = make([]*LogRecord, 0, b.maxBatch)
	}
	b.mu.Unlock()

	if shouldFlush {
		select {
		case b.flushChan <- batch:
		default:
			// Backpressure safety valve: flush inline instead of spawning
			// unbounded goroutines under sustained overload.
			b.logBackpressure()
			if err := b.store.InsertLogBatch(batch); err != nil {
				log.Printf("duckdb flush error (overflow-inline): %v", err)
			}
		}
	}
}

// Stop flushes remaining records and waits for all writes to complete.
func (b *InsertBuffer) Stop() {
	close(b.done)
	// Wait for tickLoop to finish its final drain before closing flushChan,
	// ensuring all pending records are sent to the flush channel.
	b.tickWg.Wait()
	close(b.flushChan)
	b.wg.Wait()
}

// InsertLogBatch appends a batch of raw log records into DuckDB in a single transaction.
func (s *Store) InsertLogBatch(records []*LogRecord) error {
	if len(records) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare log insert statement
	logStmt, err := tx.Prepare(`INSERT INTO logs (timestamp, orig_timestamp, level, level_num, message, raw_line, service, hostname, pid, attributes, source, app) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer logStmt.Close()

	for _, r := range records {
		// Marshal attributes to JSON
		attrsJSON := []byte("{}")
		if len(r.Attributes) > 0 {
			if data, err := json.Marshal(r.Attributes); err != nil {
				log.Printf("duckdb: failed to marshal attributes: %v", err)
			} else {
				attrsJSON = data
			}
		}

		var origTS interface{}
		if !r.OrigTimestamp.IsZero() {
			origTS = r.OrigTimestamp
		}

		app := r.App
		if app == "" {
			app = "default"
		}

		_, err := logStmt.Exec(
			r.Timestamp, origTS, r.Level, r.LevelNum,
			r.Message, r.RawLine, r.Service, r.Hostname,
			r.PID, string(attrsJSON), r.Source, app,
		)
		if err != nil {
			log.Printf("duckdb insert error (individual record): %v", err)
			continue
		}
	}

	return tx.Commit()
}
