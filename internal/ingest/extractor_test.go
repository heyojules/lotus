package ingest

import (
	"testing"
	"time"
)

func TestParseJSONLogEntry_InvalidJSON(t *testing.T) {
	t.Parallel()
	entry := ParseJSONLogEntry("this is not json")
	if entry != nil {
		t.Error("ParseJSONLogEntry should return nil for invalid JSON")
	}
}

func TestParseJSONLogEntries_NonOTELJSON(t *testing.T) {
	t.Parallel()
	entries := ParseJSONLogEntries(`{"level":"info","msg":"legacy payload"}`)
	if len(entries) != 0 {
		t.Fatalf("expected no entries for non-OTEL JSON, got %d", len(entries))
	}
}

func TestParseJSONLogEntry_OTELLogRecord(t *testing.T) {
	t.Parallel()
	line := `{"timeUnixNano":"1739876543210000000","severityText":"Warn","body":{"stringValue":"worker throttled"},"attributes":[{"key":"service.name","value":{"stringValue":"payments"}},{"key":"host.name","value":{"stringValue":"node-1"}}],"traceId":"00112233445566778899aabbccddeeff","spanId":"0011223344556677"}`

	entry := ParseJSONLogEntry(line)
	if entry == nil {
		t.Fatal("ParseJSONLogEntry returned nil")
	}
	if entry.Level != "WARN" {
		t.Fatalf("Level = %q, want WARN", entry.Level)
	}
	if entry.LevelNum != 13 {
		t.Fatalf("LevelNum = %d, want 13", entry.LevelNum)
	}
	if entry.Message != "worker throttled" {
		t.Fatalf("Message = %q, want %q", entry.Message, "worker throttled")
	}
	if entry.App != "payments" {
		t.Fatalf("App = %q, want %q", entry.App, "payments")
	}
	if entry.Attributes["service.name"] != "payments" {
		t.Fatalf("service.name attribute = %q, want %q", entry.Attributes["service.name"], "payments")
	}
	if entry.Attributes["trace.id"] == "" {
		t.Fatal("trace.id attribute should be set from OTEL traceId")
	}
	if entry.OrigTimestamp.IsZero() {
		t.Fatal("OrigTimestamp should be set from timeUnixNano")
	}
	if entry.OrigTimestamp.Year() != 2025 {
		t.Fatalf("OrigTimestamp year = %d, want 2025", entry.OrigTimestamp.Year())
	}
}

func TestParseJSONLogEntries_OTELExportEnvelope(t *testing.T) {
	t.Parallel()
	line := `{
		"resourceLogs": [
			{
				"resource": {
					"attributes": [
						{"key":"service.name","value":{"stringValue":"checkout"}},
						{"key":"host.name","value":{"stringValue":"node-a"}}
					]
				},
				"scopeLogs": [
					{
						"scope": {"name":"ingester","version":"1.2.3"},
						"logRecords": [
							{
								"timeUnixNano":"1739876543210000000",
								"severityNumber":9,
								"body":{"stringValue":"first OTEL log"},
								"attributes":[{"key":"http.method","value":{"stringValue":"GET"}}]
							},
							{
								"timeUnixNano":"1739876544210000000",
								"severityText":"Error",
								"body":{"stringValue":"second OTEL log"},
								"attributes":[{"key":"http.status_code","value":{"intValue":"503"}}]
							}
						]
					}
				]
			}
		]
	}`

	entries := ParseJSONLogEntries(line)
	if got := len(entries); got != 2 {
		t.Fatalf("record count = %d, want 2", got)
	}

	first := entries[0]
	if first.Level != "INFO" {
		t.Fatalf("first level = %q, want INFO", first.Level)
	}
	if first.LevelNum != 9 {
		t.Fatalf("first level_num = %d, want 9", first.LevelNum)
	}
	if first.Message != "first OTEL log" {
		t.Fatalf("first message = %q, want %q", first.Message, "first OTEL log")
	}
	if first.Attributes["service.name"] != "checkout" {
		t.Fatalf("first service.name = %q, want %q", first.Attributes["service.name"], "checkout")
	}
	if first.Attributes["otel.scope.name"] != "ingester" {
		t.Fatalf("first otel.scope.name = %q, want %q", first.Attributes["otel.scope.name"], "ingester")
	}
	if first.App != "checkout" {
		t.Fatalf("first app = %q, want %q", first.App, "checkout")
	}

	second := entries[1]
	if second.Level != "ERROR" {
		t.Fatalf("second level = %q, want ERROR", second.Level)
	}
	if second.LevelNum != 17 {
		t.Fatalf("second level_num = %d, want 17", second.LevelNum)
	}
	if second.Message != "second OTEL log" {
		t.Fatalf("second message = %q, want %q", second.Message, "second OTEL log")
	}
	if second.Attributes["http.status_code"] != "503" {
		t.Fatalf("second http.status_code = %q, want %q", second.Attributes["http.status_code"], "503")
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

func TestParseJSONLogEntry_OTELTimestampReceiveTime(t *testing.T) {
	t.Parallel()
	line := `{"timeUnixNano":"1739876543210000000","severityText":"Info","body":{"stringValue":"test"}}`
	entry := ParseJSONLogEntry(line)
	if entry == nil {
		t.Fatal("ParseJSONLogEntry returned nil")
	}
	if entry.OrigTimestamp.IsZero() {
		t.Fatal("OrigTimestamp should be set from OTEL timeUnixNano")
	}
	if entry.LevelNum != 9 {
		t.Fatalf("LevelNum = %d, want 9", entry.LevelNum)
	}
	if time.Since(entry.Timestamp) > 5*time.Second {
		t.Error("Timestamp (receive time) should be recent")
	}
}
