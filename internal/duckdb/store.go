package duckdb

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/control-theory/lotus/internal/duckdb/migrate"
	_ "github.com/marcboeker/go-duckdb"
)

// Store manages the DuckDB database connection and provides query methods.
type Store struct {
	db           *sql.DB
	mu           sync.RWMutex
	dbPath       string
	QueryTimeout time.Duration
	querySlots   chan struct{}
}

// NewStore opens or creates a DuckDB database.
// If dbPath is empty, an in-memory database is used.
// An optional queryTimeout can be passed; it defaults to 30s.
func NewStore(dbPath string, queryTimeout ...time.Duration) (*Store, error) {
	dsn := ""
	if dbPath != "" {
		// Ensure parent directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		dsn = dbPath
	}

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, err
	}

	// DuckDB is single-writer; limit pool to avoid contention.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)

	if err := migrate.NewRunner(db).Run(); err != nil {
		db.Close()
		return nil, err
	}

	qt := 30 * time.Second
	if len(queryTimeout) > 0 && queryTimeout[0] > 0 {
		qt = queryTimeout[0]
	}

	return &Store{
		db:           db,
		dbPath:       dbPath,
		QueryTimeout: qt,
		querySlots:   make(chan struct{}, 8),
	}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB for direct query access (e.g., AI queries).
func (s *Store) DB() *sql.DB {
	return s.db
}

// SetMaxConcurrentQueries configures global read-query concurrency.
// Values <= 0 disable the limit.
func (s *Store) SetMaxConcurrentQueries(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n <= 0 {
		s.querySlots = nil
		return
	}
	s.querySlots = make(chan struct{}, n)
}

// DeleteBefore deletes all log records with a timestamp before the given cutoff.
// Returns the number of rows deleted.
func (s *Store) DeleteBefore(cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec("DELETE FROM logs WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
