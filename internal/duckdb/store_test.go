package duckdb

import (
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore("")
	if err != nil {
		t.Fatalf("NewStore(\"\") failed: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func insertTestRecords(t *testing.T, store *Store, records []*LogRecord) {
	t.Helper()
	if err := store.InsertLogBatch(records); err != nil {
		t.Fatalf("InsertLogBatch failed: %v", err)
	}
}

func TestInsertLogBatch(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", LevelNum: 30, Message: "hello world test", Source: "stdin"},
		{Timestamp: time.Now(), Level: "ERROR", LevelNum: 50, Message: "connection failed retry", Source: "stdin"},
		{Timestamp: time.Now(), Level: "WARN", LevelNum: 40, Message: "disk usage high warning", Source: "file",
			Attributes: map[string]string{"host": "web1", "region": "us-east"}},
	}

	insertTestRecords(t, store, records)

	// Verify log count
	count, err := store.TotalLogCount(QueryOpts{})
	if err != nil {
		t.Fatalf("TotalLogCount: %v", err)
	}
	if count != 3 {
		t.Errorf("TotalLogCount = %d, want 3", count)
	}

	// Verify word counts were aggregated
	words, err := store.TopWords(10, QueryOpts{})
	if err != nil {
		t.Fatalf("TopWords: %v", err)
	}
	if len(words) == 0 {
		t.Error("TopWords returned no results after insert")
	}

	// Verify attribute counts were aggregated
	attrs, err := store.TopAttributes(10, QueryOpts{})
	if err != nil {
		t.Fatalf("TopAttributes: %v", err)
	}
	// Should have "host"="web1" and "region"="us-east"
	if len(attrs) < 2 {
		t.Errorf("TopAttributes returned %d results, want at least 2", len(attrs))
	}
}

func TestTopWords(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "request processed successfully"},
		{Timestamp: time.Now(), Level: "INFO", Message: "request processed with errors"},
		{Timestamp: time.Now(), Level: "INFO", Message: "request timeout"},
	}
	insertTestRecords(t, store, records)

	words, err := store.TopWords(5, QueryOpts{})
	if err != nil {
		t.Fatalf("TopWords: %v", err)
	}

	// "request" should appear 3 times and be the top word
	if len(words) == 0 {
		t.Fatal("TopWords returned no results")
	}
	if words[0].Word != "request" {
		t.Errorf("top word = %q, want %q", words[0].Word, "request")
	}
	if words[0].Count != 3 {
		t.Errorf("top word count = %d, want 3", words[0].Count)
	}
}

func TestTopAttributeKeys(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "test",
			Attributes: map[string]string{"env": "prod", "region": "us-east"}},
		{Timestamp: time.Now(), Level: "INFO", Message: "test",
			Attributes: map[string]string{"env": "staging", "region": "eu-west"}},
		{Timestamp: time.Now(), Level: "INFO", Message: "test",
			Attributes: map[string]string{"env": "dev"}},
	}
	insertTestRecords(t, store, records)

	keys, err := store.TopAttributeKeys(10, QueryOpts{})
	if err != nil {
		t.Fatalf("TopAttributeKeys: %v", err)
	}
	if len(keys) < 2 {
		t.Fatalf("TopAttributeKeys returned %d results, want at least 2", len(keys))
	}

	// "env" should have 3 unique values
	for _, k := range keys {
		if k.Key == "env" {
			if k.UniqueValues != 3 {
				t.Errorf("env unique values = %d, want 3", k.UniqueValues)
			}
		}
	}
}

func TestSeverityCounts(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "ok"},
		{Timestamp: time.Now(), Level: "INFO", Message: "ok"},
		{Timestamp: time.Now(), Level: "ERROR", Message: "fail"},
		{Timestamp: time.Now(), Level: "WARN", Message: "caution"},
	}
	insertTestRecords(t, store, records)

	counts, err := store.SeverityCounts(QueryOpts{})
	if err != nil {
		t.Fatalf("SeverityCounts: %v", err)
	}

	if counts["INFO"] != 2 {
		t.Errorf("INFO count = %d, want 2", counts["INFO"])
	}
	if counts["ERROR"] != 1 {
		t.Errorf("ERROR count = %d, want 1", counts["ERROR"])
	}
	if counts["WARN"] != 1 {
		t.Errorf("WARN count = %d, want 1", counts["WARN"])
	}
}

func TestTotalLogCount(t *testing.T) {
	store := newTestStore(t)

	count, err := store.TotalLogCount(QueryOpts{})
	if err != nil {
		t.Fatalf("TotalLogCount: %v", err)
	}
	if count != 0 {
		t.Errorf("empty store TotalLogCount = %d, want 0", count)
	}
}

func TestExecuteQuery_SelectAllowed(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "test log"},
	}
	insertTestRecords(t, store, records)

	results, err := store.ExecuteQuery("SELECT COUNT(*) as cnt FROM logs")
	if err != nil {
		t.Fatalf("ExecuteQuery SELECT: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ExecuteQuery returned %d rows, want 1", len(results))
	}
}

func TestExecuteQuery_WithAllowed(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "test log"},
	}
	insertTestRecords(t, store, records)

	results, err := store.ExecuteQuery("WITH c AS (SELECT COUNT(*) AS cnt FROM logs) SELECT cnt FROM c")
	if err != nil {
		t.Fatalf("ExecuteQuery WITH: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ExecuteQuery WITH returned %d rows, want 1", len(results))
	}
}

func TestExecuteQuery_DMLRejected(t *testing.T) {
	store := newTestStore(t)

	rejected := []string{
		"INSERT INTO logs (level, message) VALUES ('INFO', 'hack')",
		"UPDATE logs SET message = 'hacked'",
		"DELETE FROM logs",
		"DROP TABLE logs",
		"CREATE TABLE evil (id int)",
		"ALTER TABLE logs ADD COLUMN evil varchar",
		"TRUNCATE logs",
	}

	for _, sql := range rejected {
		_, err := store.ExecuteQuery(sql)
		if err == nil {
			t.Errorf("ExecuteQuery(%q) should have been rejected", sql)
		}
	}
}

func TestExecuteQuery_DuckDBKeywordsRejected(t *testing.T) {
	store := newTestStore(t)

	// Test keyword rejection without semicolons (keyword denylist).
	rejected := []struct {
		sql     string
		keyword string
	}{
		{"SELECT COPY(logs, '/tmp/dump.csv') FROM logs", "COPY"},
		{"SELECT ATTACH FROM logs", "ATTACH"},
		{"SELECT LOAD FROM logs", "LOAD"},
		{"SELECT EXPORT FROM logs", "EXPORT"},
		{"SELECT IMPORT FROM logs", "IMPORT"},
		{"SELECT INSTALL FROM logs", "INSTALL"},
		{"SELECT CALL FROM logs", "CALL"},
		{"SELECT EXECUTE FROM logs", "EXECUTE"},
		{"SELECT PRAGMA FROM logs", "PRAGMA"},
		{"SELECT SET FROM logs", "SET"},
	}

	for _, tt := range rejected {
		_, err := store.ExecuteQuery(tt.sql)
		if err == nil {
			t.Errorf("ExecuteQuery should reject %s keyword", tt.keyword)
		}
		if err != nil && !strings.Contains(err.Error(), tt.keyword) {
			t.Errorf("ExecuteQuery error %q should mention keyword %s", err.Error(), tt.keyword)
		}
	}

	// Test semicolon rejection (prevents statement chaining).
	semicolonCases := []string{
		"SELECT * FROM logs; DROP TABLE logs",
		"SELECT * FROM logs; COPY logs TO '/tmp/dump.csv'",
	}
	for _, sql := range semicolonCases {
		_, err := store.ExecuteQuery(sql)
		if err == nil {
			t.Errorf("ExecuteQuery should reject query with semicolons: %s", sql)
		}
		if err != nil && !strings.Contains(err.Error(), "semicolons") {
			t.Errorf("ExecuteQuery error %q should mention semicolons", err.Error())
		}
	}
}

func TestTableRowCounts(t *testing.T) {
	store := newTestStore(t)

	counts, err := store.TableRowCounts()
	if err != nil {
		t.Fatalf("TableRowCounts: %v", err)
	}

	// Logs table should be present
	for _, table := range []string{"logs"} {
		if _, ok := counts[table]; !ok {
			t.Errorf("TableRowCounts missing table %q", table)
		}
	}
}
