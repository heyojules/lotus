package logparse

import "testing"

func TestNormalizeSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Standard forms
		{"TRACE", "TRACE"}, {"DEBUG", "DEBUG"}, {"INFO", "INFO"},
		{"WARN", "WARN"}, {"ERROR", "ERROR"}, {"FATAL", "FATAL"},
		// Variants
		{"TRAC", "TRACE"}, {"TRC", "TRACE"},
		{"DEBU", "DEBUG"}, {"DBG", "DEBUG"}, {"DEB", "DEBUG"},
		{"INFORMATION", "INFO"}, {"INF", "INFO"},
		{"WARNING", "WARN"}, {"WRNG", "WARN"}, {"WRN", "WARN"},
		{"ERR", "ERROR"}, {"ERRO", "ERROR"},
		{"FATL", "FATAL"}, {"FTL", "FATAL"},
		{"CRITICAL", "FATAL"}, {"CRIT", "FATAL"}, {"CRT", "FATAL"},
		{"PANIC", "FATAL"}, {"PNC", "FATAL"},
		// Case insensitive
		{"info", "INFO"}, {"warn", "WARN"}, {"error", "ERROR"},
		{"debug", "DEBUG"}, {"trace", "TRACE"}, {"fatal", "FATAL"},
		// Prefix matching
		{"INFORMATION_EXTRA", "INFO"}, {"WARNING_LEVEL", "WARN"},
		{"ERROR_CODE_42", "ERROR"}, {"DEBUG_VERBOSE", "DEBUG"},
		{"TRACE_ALL", "TRACE"}, {"FATAL_CRASH", "FATAL"},
		{"CRITICAL_ALERT", "FATAL"},
		// Unknown defaults to INFO
		{"", "INFO"}, {"UNKNOWN", "INFO"}, {"foo", "INFO"},
		// Whitespace
		{"  INFO  ", "INFO"}, {"\tWARN\t", "WARN"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeSeverity(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeSeverity(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractSeverityFromText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2024-01-01 INFO Starting server", "INFO"},
		{"ERROR: connection refused", "ERROR"},
		{"[WARN] disk usage high", "WARN"},
		{"FATAL out of memory", "FATAL"},
		{"DEBUG checking cache", "DEBUG"},
		{"TRACE entering function", "TRACE"},
		{"WARNING deprecated API", "WARN"},
		{"CRITICAL system failure", "FATAL"},
		{"no severity here", "INFO"},
		{"", "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExtractSeverityFromText(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractSeverityFromText(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
