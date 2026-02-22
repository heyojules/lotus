package duckdb

import (
	"testing"
	"time"
)

func TestListApps(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "hello from api", App: "api"},
		{Timestamp: time.Now(), Level: "WARN", Message: "worker slow", App: "worker"},
		{Timestamp: time.Now(), Level: "INFO", Message: "default log", App: "default"},
		{Timestamp: time.Now(), Level: "ERROR", Message: "api error", App: "api"},
	}
	insertTestRecords(t, store, records)

	apps, err := store.ListApps()
	if err != nil {
		t.Fatalf("ListApps: %v", err)
	}
	if len(apps) != 3 {
		t.Fatalf("ListApps returned %d apps, want 3; got %v", len(apps), apps)
	}
	// Should be sorted: api, default, worker
	expected := []string{"api", "default", "worker"}
	for i, want := range expected {
		if apps[i] != want {
			t.Errorf("apps[%d] = %q, want %q", i, apps[i], want)
		}
	}
}

func TestTotalLogCountByApp(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "a", App: "api"},
		{Timestamp: time.Now(), Level: "INFO", Message: "b", App: "api"},
		{Timestamp: time.Now(), Level: "INFO", Message: "c", App: "worker"},
		{Timestamp: time.Now(), Level: "INFO", Message: "d", App: "default"},
	}
	insertTestRecords(t, store, records)

	count, err := store.TotalLogCount(QueryOpts{App: "api"})
	if err != nil {
		t.Fatalf("TotalLogCount(api): %v", err)
	}
	if count != 2 {
		t.Errorf("api count = %d, want 2", count)
	}

	count, err = store.TotalLogCount(QueryOpts{App: "worker"})
	if err != nil {
		t.Fatalf("TotalLogCount(worker): %v", err)
	}
	if count != 1 {
		t.Errorf("worker count = %d, want 1", count)
	}

	count, err = store.TotalLogCount(QueryOpts{App: "nonexistent"})
	if err != nil {
		t.Fatalf("TotalLogCount(nonexistent): %v", err)
	}
	if count != 0 {
		t.Errorf("nonexistent count = %d, want 0", count)
	}
}

func TestTopWordsByApp(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "request processed successfully", App: "api"},
		{Timestamp: time.Now(), Level: "INFO", Message: "request processed with errors", App: "api"},
		{Timestamp: time.Now(), Level: "INFO", Message: "background job completed", App: "worker"},
	}
	insertTestRecords(t, store, records)

	words, err := store.TopWords(5, QueryOpts{App: "api"})
	if err != nil {
		t.Fatalf("TopWords(api): %v", err)
	}
	if len(words) == 0 {
		t.Fatal("TopWords returned no results for api")
	}
	// "request" should appear twice in api logs
	found := false
	for _, w := range words {
		if w.Word == "request" {
			found = true
			if w.Count != 2 {
				t.Errorf("request count = %d, want 2", w.Count)
			}
		}
	}
	if !found {
		t.Error("expected 'request' in TopWords results for api")
	}

	// Worker should not have "request"
	words, err = store.TopWords(5, QueryOpts{App: "worker"})
	if err != nil {
		t.Fatalf("TopWords(worker): %v", err)
	}
	for _, w := range words {
		if w.Word == "request" {
			t.Error("worker should not have 'request' word")
		}
	}
}

func TestTopAttributesByApp(t *testing.T) {
	store := newTestStore(t)

	records := []*LogRecord{
		{
			Timestamp:  time.Now(),
			Level:      "INFO",
			Message:    "api log 1",
			App:        "api",
			Attributes: map[string]string{"env": "prod", "host": "api-1"},
		},
		{
			Timestamp:  time.Now(),
			Level:      "INFO",
			Message:    "api log 2",
			App:        "api",
			Attributes: map[string]string{"env": "prod", "host": "api-2"},
		},
		{
			Timestamp:  time.Now(),
			Level:      "INFO",
			Message:    "worker log",
			App:        "worker",
			Attributes: map[string]string{"env": "dev", "host": "worker-1"},
		},
	}
	insertTestRecords(t, store, records)

	attrs, err := store.TopAttributes(10, QueryOpts{App: "api"})
	if err != nil {
		t.Fatalf("TopAttributes(api): %v", err)
	}
	if len(attrs) == 0 {
		t.Fatal("TopAttributes returned no rows for api")
	}

	foundEnvProd := false
	for _, attr := range attrs {
		if attr.Key == "env" && attr.Value == "prod" {
			foundEnvProd = true
			if attr.Count != 2 {
				t.Errorf("env=prod count = %d, want 2", attr.Count)
			}
		}
		if attr.Key == "env" && attr.Value == "dev" {
			t.Fatal("api-scoped query should not include worker env=dev")
		}
	}
	if !foundEnvProd {
		t.Fatal("expected env=prod in app-scoped attributes")
	}
}

func TestListApps_Empty(t *testing.T) {
	store := newTestStore(t)

	apps, err := store.ListApps()
	if err != nil {
		t.Fatalf("ListApps: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("ListApps on empty store returned %d, want 0", len(apps))
	}
}

func TestInsertLogBatch_DefaultApp(t *testing.T) {
	store := newTestStore(t)

	// Insert record with empty App — should default to "default"
	records := []*LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "no app set"},
	}
	insertTestRecords(t, store, records)

	apps, err := store.ListApps()
	if err != nil {
		t.Fatalf("ListApps: %v", err)
	}
	if len(apps) != 1 || apps[0] != "default" {
		t.Errorf("expected [default], got %v", apps)
	}
}
