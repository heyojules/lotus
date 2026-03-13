package ingest

import (
	"strings"

	"github.com/tinytelemetry/tiny-telemetry/internal/logparse"
)

// ExtractService extracts the service name from log attributes.
func ExtractService(attributes map[string]string) string {
	for _, key := range []string{"service.name", "service", "serviceName", "app", "name"} {
		if v := attributes[key]; v != "" {
			return v
		}
	}
	return "unknown"
}

// ExtractApp extracts the application name from log attributes.
func ExtractApp(attributes map[string]string) string {
	for _, key := range []string{"app", "service.name", "service_name", "service", "name"} {
		if v := attributes[key]; v != "" {
			return v
		}
	}
	return ""
}

// ExtractHostname extracts the hostname from log attributes.
func ExtractHostname(attributes map[string]string) string {
	for _, key := range []string{"host", "hostname", "host.name"} {
		if v := attributes[key]; v != "" {
			return v
		}
	}
	return ""
}

// SeverityFromNumber maps an OTEL severity number to its text representation.
func SeverityFromNumber(number int) string {
	switch {
	case number >= 1 && number <= 4:
		return "TRACE"
	case number >= 5 && number <= 8:
		return "DEBUG"
	case number >= 9 && number <= 12:
		return "INFO"
	case number >= 13 && number <= 16:
		return "WARN"
	case number >= 17 && number <= 20:
		return "ERROR"
	case number >= 21 && number <= 24:
		return "FATAL"
	default:
		return ""
	}
}

// DefaultSeverityNumber returns the canonical OTEL severity number for a normalized level string.
func DefaultSeverityNumber(level string) int {
	switch logparse.NormalizeSeverity(level) {
	case "TRACE":
		return 1
	case "DEBUG":
		return 5
	case "INFO":
		return 9
	case "WARN":
		return 13
	case "ERROR":
		return 17
	case "FATAL":
		return 21
	default:
		return 9
	}
}

// SanitizeMessage replaces control characters (tabs, newlines) with spaces.
func SanitizeMessage(message string) string {
	clean := strings.ReplaceAll(message, "\t", " ")
	clean = strings.ReplaceAll(clean, "\n", " ")
	clean = strings.ReplaceAll(clean, "\r", " ")
	return clean
}

// CloneAttributes returns a shallow copy of the attribute map.
func CloneAttributes(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// MergeAttributes copies src entries into dst, overwriting on conflict.
func MergeAttributes(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}
