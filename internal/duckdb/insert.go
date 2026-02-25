package duckdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/control-theory/lotus/internal/journal"
	"github.com/control-theory/lotus/internal/model"
)

// DefaultFlushQueueSize is the number of batches that can be queued for async flushing.
const DefaultFlushQueueSize = 64

var eventIDCounter atomic.Uint64

type journaledRecord struct {
	seq    uint64
	record *LogRecord
}

type durableJournal interface {
	Append(record *model.LogRecord) (uint64, error)
	Commit(seq uint64) error
	Close() error
}

// InsertBuffer batches log records and flushes them to DuckDB asynchronously.
// Add() never blocks on DuckDB writes - records are sent to a flush goroutine.
type InsertBuffer struct {
	writer        model.LogWriter
	mu            sync.Mutex
	pending       []journaledRecord
	flushChan     chan []journaledRecord // async flush queue
	maxBatch      int
	flushInterval time.Duration
	done          chan struct{}
	wg            sync.WaitGroup
	tickWg        sync.WaitGroup // separate WaitGroup for tickLoop
	journal       durableJournal

	// backpressureCount tracks inline flushes for throttled logging.
	backpressureCount atomic.Int64
	lastBPLog         atomic.Int64 // unix timestamp of last backpressure log
}

// InsertBufferConfig holds tunable parameters for the insert buffer.
type InsertBufferConfig struct {
	BatchSize      int
	FlushInterval  time.Duration
	FlushQueueSize int
	Journal        *journal.Journal
}

// NewInsertBuffer creates a new insert buffer that flushes to the store.
// The flush goroutine processes batches asynchronously so Add() never blocks on IO.
func NewInsertBuffer(writer model.LogWriter, conf ...InsertBufferConfig) *InsertBuffer {
	batchSize := 2000
	flushInterval := 100 * time.Millisecond
	flushQueueSize := DefaultFlushQueueSize
	if len(conf) > 0 {
		if conf[0].BatchSize > 0 {
			batchSize = conf[0].BatchSize
		}
		if conf[0].FlushInterval > 0 {
			flushInterval = conf[0].FlushInterval
		}
		if conf[0].FlushQueueSize > 0 {
			flushQueueSize = conf[0].FlushQueueSize
		}
	}

	b := &InsertBuffer{
		writer:        writer,
		pending:       make([]journaledRecord, 0, batchSize),
		flushChan:     make(chan []journaledRecord, flushQueueSize),
		maxBatch:      batchSize,
		flushInterval: flushInterval,
		done:          make(chan struct{}),
	}
	if len(conf) > 0 && conf[0].Journal != nil {
		b.journal = conf[0].Journal
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
	b.pending = make([]journaledRecord, 0, b.maxBatch)
	b.mu.Unlock()

	// Non-blocking send to flush channel. If channel is full, flush synchronously
	// as a safety valve (this means DuckDB is falling behind).
	select {
	case b.flushChan <- batch:
	default:
		b.logBackpressure()
		if err := b.flushBatch(batch); err != nil {
			log.Printf("duckdb flush error (inline): %v", err)
		}
	}
}

// flushWorker processes batches from the flush channel.
func (b *InsertBuffer) flushWorker() {
	defer b.wg.Done()
	for batch := range b.flushChan {
		if err := b.flushBatch(batch); err != nil {
			log.Printf("duckdb flush error: %v", err)
		}
	}
}

// Add queues a record for batch insertion. This never blocks on DuckDB IO.
func (b *InsertBuffer) Add(record *LogRecord) {
	if record.EventID == "" {
		record.EventID = nextEventID()
	}

	seq := uint64(0)
	if b.journal != nil {
		for {
			var err error
			seq, err = b.journal.Append(record)
			if err == nil {
				break
			}
			log.Printf("duckdb: journal append failed, retrying: %v", err)
			select {
			case <-b.done:
				return
			case <-time.After(200 * time.Millisecond):
			}
		}
	}

	b.mu.Lock()
	b.pending = append(b.pending, journaledRecord{
		seq:    seq,
		record: record,
	})
	shouldFlush := len(b.pending) >= b.maxBatch
	var batch []journaledRecord
	if shouldFlush {
		batch = b.pending
		b.pending = make([]journaledRecord, 0, b.maxBatch)
	}
	b.mu.Unlock()

	if shouldFlush {
		select {
		case b.flushChan <- batch:
		default:
			// Backpressure safety valve: flush inline instead of spawning
			// unbounded goroutines under sustained overload.
			b.logBackpressure()
			if err := b.flushBatch(batch); err != nil {
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
	if b.journal != nil {
		if err := b.journal.Close(); err != nil {
			log.Printf("duckdb: journal close error: %v", err)
		}
	}
}

func (b *InsertBuffer) flushBatch(batch []journaledRecord) error {
	if len(batch) == 0 {
		return nil
	}

	records := make([]*LogRecord, 0, len(batch))
	for _, item := range batch {
		records = append(records, item.record)
	}

	if err := b.writer.InsertLogBatch(records); err != nil {
		return err
	}

	if b.journal != nil {
		maxSeq := uint64(0)
		for _, item := range batch {
			if item.seq > maxSeq {
				maxSeq = item.seq
			}
		}
		if maxSeq > 0 {
			if err := b.journal.Commit(maxSeq); err != nil {
				return fmt.Errorf("journal commit seq=%d: %w", maxSeq, err)
			}
		}
	}
	return nil
}

// InsertLogBatch appends a batch of raw log records into DuckDB in a single transaction.
// If any individual record fails to insert, the entire batch is rolled back and retried
// record-by-record to salvage as many records as possible.
func (s *Store) InsertLogBatch(records []*LogRecord) error {
	if len(records) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.QueryTimeout)
	defer cancel()

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.insertBatchTx(ctx, records)
	if err == nil {
		return nil
	}

	// Batch failed — retry record-by-record to salvage what we can.
	var failed int
	for _, r := range records {
		if rerr := s.insertBatchTx(ctx, []*LogRecord{r}); rerr != nil {
			failed++
			log.Printf("duckdb: dropping record (service=%s msg=%.80s): %v", r.Service, r.Message, rerr)
		}
	}
	if failed > 0 {
		log.Printf("duckdb: batch partially failed — %d/%d records dropped", failed, len(records))
	}
	return nil
}

// insertBatchTx inserts records in a single transaction.
func (s *Store) insertBatchTx(ctx context.Context, records []*LogRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	logStmt, err := tx.PrepareContext(ctx, `INSERT INTO logs (timestamp, orig_timestamp, level, level_num, message, raw_line, service, hostname, pid, attributes, source, app, event_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer logStmt.Close()

	for _, r := range records {
		attrsJSON := []byte("{}")
		if len(r.Attributes) > 0 {
			if data, merr := json.Marshal(r.Attributes); merr != nil {
				log.Printf("duckdb: failed to marshal attributes, using empty: %v", merr)
			} else {
				attrsJSON = data
			}
		}

		var origTS any
		if !r.OrigTimestamp.IsZero() {
			origTS = r.OrigTimestamp
		}

		app := r.App
		if app == "" {
			app = "default"
		}
		eventID := r.EventID
		if eventID == "" {
			eventID = nextEventID()
		}

		if _, err := logStmt.ExecContext(
			ctx,
			r.Timestamp, origTS, r.Level, r.LevelNum,
			r.Message, r.RawLine, r.Service, r.Hostname,
			r.PID, string(attrsJSON), r.Source, app, eventID,
		); err != nil {
			return fmt.Errorf("record insert: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func nextEventID() string {
	n := eventIDCounter.Add(1)
	return fmt.Sprintf("%x-%x", time.Now().UTC().UnixNano(), n)
}
