package tui

import (
	"testing"

	"github.com/control-theory/lotus/internal/model"
)

func TestNormalizeSeverityLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"TRACE", "TRACE"},
		{"TRC", "TRACE"},
		{"DEBUG", "DEBUG"},
		{"DBG", "DEBUG"},
		{"DEBG", "DEBUG"},
		{"INFO", "INFO"},
		{"INFORMATION", "INFO"},
		{"INF", "INFO"},
		{"WARN", "WARN"},
		{"WARNING", "WARN"},
		{"WRN", "WARN"},
		{"ERROR", "ERROR"},
		{"ERR", "ERROR"},
		{"FATAL", "FATAL"},
		{"FTL", "FATAL"},
		{"CRITICAL", "CRITICAL"},
		{"CRIT", "CRITICAL"},
		{"CRT", "CRITICAL"},
		{"", "UNKNOWN"},
		{"something", "UNKNOWN"},
		{"  info  ", "INFO"}, // whitespace
		{"error", "ERROR"},   // lowercase
		{"Error", "ERROR"},   // mixed case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := normalizeSeverityLevel(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeSeverityLevel(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSeverityCounts_AddCount(t *testing.T) {
	t.Parallel()
	sc := &SeverityCounts{}

	sc.AddCount("INFO")
	sc.AddCount("INFO")
	sc.AddCount("ERROR")
	sc.AddCount("warning")   // lowercase, should normalize to WARN
	sc.AddCount("something") // unknown

	if sc.Info != 2 {
		t.Errorf("Info = %d, want 2", sc.Info)
	}
	if sc.Error != 1 {
		t.Errorf("Error = %d, want 1", sc.Error)
	}
	if sc.Warn != 1 {
		t.Errorf("Warn = %d, want 1", sc.Warn)
	}
	if sc.Unknown != 1 {
		t.Errorf("Unknown = %d, want 1", sc.Unknown)
	}
	if sc.Total != 5 {
		t.Errorf("Total = %d, want 5", sc.Total)
	}
}

func TestNewSeverityCountsFromEntries(t *testing.T) {
	t.Parallel()
	entries := []model.LogRecord{
		{Level: "INFO"},
		{Level: "ERROR"},
		{Level: "INFO"},
		{Level: "DEBUG"},
	}

	counts := NewSeverityCountsFromEntries(entries)
	if counts.Info != 2 {
		t.Errorf("Info = %d, want 2", counts.Info)
	}
	if counts.Error != 1 {
		t.Errorf("Error = %d, want 1", counts.Error)
	}
	if counts.Debug != 1 {
		t.Errorf("Debug = %d, want 1", counts.Debug)
	}
	if counts.Total != 4 {
		t.Errorf("Total = %d, want 4", counts.Total)
	}
}
