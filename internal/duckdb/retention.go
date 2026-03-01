package duckdb

import (
	"log"
	"sync"
	"time"
)

// RetentionConfig holds configuration for the retention cleaner.
type RetentionConfig struct {
	RetentionDays int
}

// RetentionCleaner periodically deletes logs older than the configured retention period.
type RetentionCleaner struct {
	store         *Store
	retentionDays int
	done          chan struct{}
	wg            sync.WaitGroup
	tickWg        sync.WaitGroup
	stopOnce      sync.Once
}

// NewRetentionCleaner creates a retention cleaner that deletes expired logs.
// Returns nil when retention is 0 (disabled).
func NewRetentionCleaner(store *Store, conf ...RetentionConfig) *RetentionCleaner {
	days := 30
	if len(conf) > 0 {
		days = conf[0].RetentionDays
	}
	if days <= 0 {
		return nil
	}

	rc := &RetentionCleaner{
		store:         store,
		retentionDays: days,
		done:          make(chan struct{}),
	}

	// Startup cleanup to catch up after downtime.
	rc.cleanup()

	rc.wg.Add(1)
	rc.tickWg.Add(1)
	go rc.tickLoop()

	return rc
}

func (rc *RetentionCleaner) tickLoop() {
	defer rc.wg.Done()
	defer rc.tickWg.Done()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rc.cleanup()
		case <-rc.done:
			return
		}
	}
}

func (rc *RetentionCleaner) cleanup() {
	cutoff := time.Now().Add(-time.Duration(rc.retentionDays) * 24 * time.Hour)

	rows, err := rc.store.DeleteBefore(cutoff)
	if err != nil {
		log.Printf("duckdb: retention cleanup error: %v", err)
		return
	}
	if rows > 0 {
		log.Printf("duckdb: retention cleanup deleted %d expired logs (older than %d days)", rows, rc.retentionDays)
	}
}

// Stop signals the cleaner to stop and waits for it to finish.
func (rc *RetentionCleaner) Stop() {
	rc.stopOnce.Do(func() {
		close(rc.done)
		rc.tickWg.Wait()
		rc.wg.Wait()
	})
}
