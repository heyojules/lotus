package migrate

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRunAppliesAllMigrations(t *testing.T) {
	db := openTestDB(t)
	r := NewRunner(db)

	if err := r.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify core tables exist by querying them
	for _, table := range []string{"logs", "schema_migrations"} {
		var name string
		err := db.QueryRow("SELECT table_name FROM information_schema.tables WHERE table_name = ?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestRunIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	r := NewRunner(db)

	if err := r.Run(); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if err := r.Run(); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	cur, pending, err := r.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if cur != 4 || pending != 0 {
		t.Errorf("expected version=4 pending=0, got version=%d pending=%d", cur, pending)
	}
}

func TestStatusReportsCorrectly(t *testing.T) {
	db := openTestDB(t)
	r := NewRunner(db)

	// Before any migration
	cur, pending, err := r.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if cur != 0 || pending != 4 {
		t.Errorf("before run: expected version=0 pending=4, got version=%d pending=%d", cur, pending)
	}

	// After running
	if err := r.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	cur, pending, err = r.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if cur != 4 || pending != 0 {
		t.Errorf("after run: expected version=4 pending=0, got version=%d pending=%d", cur, pending)
	}
}
