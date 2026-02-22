package migrate

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Runner applies versioned SQL migrations to a DuckDB database.
type Runner struct{ db *sql.DB }

// NewRunner creates a migration runner for the given database connection.
func NewRunner(db *sql.DB) *Runner {
	return &Runner{db: db}
}

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("reading embedded migrations: %w", err)
	}

	var migs []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		ver, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("parsing version from %s: %w", e.Name(), err)
		}
		data, err := migrations.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		migs = append(migs, migration{version: ver, name: e.Name(), sql: string(data)})
	}

	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })
	return migs, nil
}

func (r *Runner) bootstrap() error {
	_, err := r.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       VARCHAR NOT NULL,
		applied_at TIMESTAMP DEFAULT current_timestamp
	)`)
	return err
}

func (r *Runner) appliedVersion() (int, error) {
	var v sql.NullInt64
	err := r.db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&v)
	if err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

// Run applies all pending migrations in order. Each migration is executed
// in a transaction and recorded in the schema_migrations table.
func (r *Runner) Run() error {
	if err := r.bootstrap(); err != nil {
		return fmt.Errorf("bootstrap schema_migrations: %w", err)
	}

	migs, err := loadMigrations()
	if err != nil {
		return err
	}

	current, err := r.appliedVersion()
	if err != nil {
		return fmt.Errorf("reading applied version: %w", err)
	}

	for _, m := range migs {
		if m.version <= current {
			continue
		}
		tx, err := r.db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", m.name, err)
		}
		if _, err := tx.Exec(m.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("executing %s: %w", m.name, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", m.version, m.name); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", m.name, err)
		}
	}

	return nil
}

// Status returns the current applied version and count of pending migrations.
func (r *Runner) Status() (current int, pending int, err error) {
	if err = r.bootstrap(); err != nil {
		return 0, 0, fmt.Errorf("bootstrap schema_migrations: %w", err)
	}

	current, err = r.appliedVersion()
	if err != nil {
		return 0, 0, fmt.Errorf("reading applied version: %w", err)
	}

	migs, err := loadMigrations()
	if err != nil {
		return 0, 0, err
	}

	for _, m := range migs {
		if m.version > current {
			pending++
		}
	}

	return current, pending, nil
}
