package duckdb

import "testing"

func TestRetentionCleaner_StopIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	cleaner := NewRetentionCleaner(store, RetentionConfig{RetentionDays: 1})
	if cleaner == nil {
		t.Fatal("expected non-nil retention cleaner")
	}

	cleaner.Stop()
	cleaner.Stop()
}
