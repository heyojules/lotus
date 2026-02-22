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
