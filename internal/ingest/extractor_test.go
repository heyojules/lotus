package ingest

import (
	"testing"
	"time"
)

func TestParseJSONLogEntry_Pino(t *testing.T) {
	t.Parallel()
	line := `{"level":30,"time":1705312245000,"msg":"request processed","hostname":"web1","pid":1234,"reqId":"abc"}`
	entry := ParseJSONLogEntry(line)
	if entry == nil {
		t.Fatal("ParseJSONLogEntry returned nil for pino format")
	}
	if entry.Level != "INFO" {
		t.Errorf("severity = %q, want INFO (pino level 30)", entry.Level)
	}
	if entry.Message != "request processed" {
		t.Errorf("message = %q, want 'request processed'", entry.Message)
	}
	if entry.Attributes["hostname"] != "web1" {
		t.Errorf("hostname attr = %q, want 'web1'", entry.Attributes["hostname"])
	}
}

func TestParseJSONLogEntry_Winston(t *testing.T) {
	t.Parallel()
	line := `{"level":"error","message":"connection refused","timestamp":"2024-01-15T10:30:45.000Z","service":"api"}`
	entry := ParseJSONLogEntry(line)
	if entry == nil {
		t.Fatal("ParseJSONLogEntry returned nil for winston format")
	}
	if entry.Level != "ERROR" {
		t.Errorf("severity = %q, want ERROR", entry.Level)
	}
	if entry.Message != "connection refused" {
		t.Errorf("message = %q, want 'connection refused'", entry.Message)
	}
}

func TestParseJSONLogEntry_InvalidJSON(t *testing.T) {
	t.Parallel()
	entry := ParseJSONLogEntry("this is not json")
	if entry != nil {
		t.Error("ParseJSONLogEntry should return nil for invalid JSON")
	}
}

func TestParseJSONLogEntry_AppField(t *testing.T) {
	t.Parallel()
	line := `{"msg":"test","_app":"my-api","level":"info"}`
	entry := ParseJSONLogEntry(line)
	if entry == nil {
		t.Fatal("ParseJSONLogEntry returned nil")
	}
	if entry.App != "my-api" {
		t.Errorf("App = %q, want %q", entry.App, "my-api")
	}
	if _, exists := entry.Attributes["_app"]; exists {
		t.Error("_app should not appear in attributes")
	}
}

func TestCreateFallbackLogEntry(t *testing.T) {
	t.Parallel()
	entry := CreateFallbackLogEntry("2024-01-15 ERROR: connection refused")
	if entry == nil {
		t.Fatal("CreateFallbackLogEntry returned nil")
	}
	if entry.Level != "ERROR" {
		t.Errorf("severity = %q, want ERROR", entry.Level)
	}
	if entry.App != "default" {
		t.Errorf("App = %q, want %q", entry.App, "default")
	}
}

func TestCreateFallbackLogEntry_CleansTabs(t *testing.T) {
	t.Parallel()
	entry := CreateFallbackLogEntry("message\twith\ttabs\nand\nnewlines")
	if entry == nil {
		t.Fatal("CreateFallbackLogEntry returned nil")
	}
	if entry.Message != "message with tabs and newlines" {
		t.Errorf("message = %q, should have tabs/newlines replaced", entry.Message)
	}
}

func TestExtractStringField(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"msg":     "hello",
		"message": "world",
		"count":   float64(42),
	}

	if got := ExtractStringField(raw, "msg"); got != "hello" {
		t.Errorf("ExtractStringField(msg) = %q, want 'hello'", got)
	}
	if got := ExtractStringField(raw, "missing", "msg", "message"); got != "hello" {
		t.Errorf("ExtractStringField(missing,msg,message) = %q, want 'hello'", got)
	}
	if got := ExtractStringField(raw, "count"); got != "42" {
		t.Errorf("ExtractStringField(count) = %q, want '42'", got)
	}
	if got := ExtractStringField(raw, "nonexistent"); got != "" {
		t.Errorf("ExtractStringField(nonexistent) = %q, want empty", got)
	}
}

func TestExtractLevelFromJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		raw      map[string]interface{}
		expected string
	}{
		{"string level", map[string]interface{}{"level": "error"}, "error"},
		{"numeric level 30", map[string]interface{}{"level": float64(30)}, "INFO"},
		{"numeric level 50", map[string]interface{}{"level": float64(50)}, "ERROR"},
		{"severity field", map[string]interface{}{"severity": "warning"}, "warning"},
		{"no level", map[string]interface{}{"msg": "test"}, "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ExtractLevelFromJSON(tt.raw); got != tt.expected {
				t.Errorf("ExtractLevelFromJSON = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractTimestampFromJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  map[string]interface{}
		zero bool
		year int
	}{
		{"RFC3339", map[string]interface{}{"timestamp": "2024-01-15T10:30:45Z"}, false, 2024},
		{"unix seconds", map[string]interface{}{"ts": float64(946684800)}, false, 2000},
		{"no timestamp", map[string]interface{}{"other": "value"}, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := ExtractTimestampFromJSON(tt.raw)
			if tt.zero && !ts.IsZero() {
				t.Errorf("expected zero time, got %v", ts)
			}
			if !tt.zero && ts.IsZero() {
				t.Error("expected non-zero time")
			}
			if !tt.zero && tt.year != 0 && ts.Year() != tt.year {
				t.Errorf("year = %d, want %d", ts.Year(), tt.year)
			}
		})
	}
}

func TestExtractService(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		attrs    map[string]string
		expected string
	}{
		{"service.name", map[string]string{"service.name": "api"}, "api"},
		{"service", map[string]string{"service": "web"}, "web"},
		{"serviceName", map[string]string{"serviceName": "auth"}, "auth"},
		{"app", map[string]string{"app": "myapp"}, "myapp"},
		{"name", map[string]string{"name": "svc"}, "svc"},
		{"unknown", map[string]string{"foo": "bar"}, "unknown"},
		{"empty", map[string]string{}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ExtractService(tt.attrs); got != tt.expected {
				t.Errorf("ExtractService = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCountJSONDepth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line     string
		expected int
	}{
		{`{`, 1},
		{`}`, -1},
		{`{"key": "value"}`, 0},
		{`{"nested": {`, 2},
		{`"key": "val with { brace"`, 0}, // braces inside strings
		{`}}`, -2},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			if got := CountJSONDepth(tt.line); got != tt.expected {
				t.Errorf("CountJSONDepth(%q) = %d, want %d", tt.line, got, tt.expected)
			}
		})
	}
}

func TestParseJSONLogEntry_OrigTimestamp(t *testing.T) {
	t.Parallel()
	line := `{"msg":"test","time":"2024-01-15T10:30:45Z"}`
	entry := ParseJSONLogEntry(line)
	if entry == nil {
		t.Fatal("ParseJSONLogEntry returned nil")
	}
	if entry.OrigTimestamp.IsZero() {
		t.Error("OrigTimestamp should be set from JSON time field")
	}
	if entry.OrigTimestamp.Year() != 2024 {
		t.Errorf("OrigTimestamp year = %d, want 2024", entry.OrigTimestamp.Year())
	}
	if time.Since(entry.Timestamp) > 5*time.Second {
		t.Error("Timestamp (receive time) should be recent")
	}
}
