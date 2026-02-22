package logparse

import (
	"regexp"
	"strings"
)

// SeverityRegex matches common severity levels in log text.
var SeverityRegex = regexp.MustCompile(`(?i)\b(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)\b`)

// NormalizeSeverity converts various severity level formats to consistent all caps short forms.
func NormalizeSeverity(severity string) string {
	normalized := strings.ToUpper(strings.TrimSpace(severity))

	switch normalized {
	case "TRACE", "TRAC", "TRC":
		return "TRACE"
	case "DEBUG", "DEBU", "DBG", "DEB":
		return "DEBUG"
	case "INFO", "INFORMATION", "INF":
		return "INFO"
	case "WARN", "WARNING", "WRNG", "WRN":
		return "WARN"
	case "ERROR", "ERR", "ERRO":
		return "ERROR"
	case "FATAL", "FATL", "FTL", "CRITICAL", "CRIT", "CRT":
		return "FATAL"
	case "PANIC", "PNC":
		return "FATAL"
	default:
		if len(normalized) >= 4 {
			prefix := normalized[:4]
			switch prefix {
			case "INFO":
				return "INFO"
			case "WARN":
				return "WARN"
			case "ERRO":
				return "ERROR"
			case "DEBU":
				return "DEBUG"
			case "TRAC":
				return "TRACE"
			case "FATA", "CRIT":
				return "FATAL"
			}
		}
		return "INFO"
	}
}

// ExtractSeverityFromText extracts severity level from log message text.
func ExtractSeverityFromText(message string) string {
	matches := SeverityRegex.FindStringSubmatch(message)
	if len(matches) > 1 {
		severity := strings.ToUpper(matches[1])
		switch severity {
		case "WARNING":
			return "WARN"
		case "CRITICAL":
			return "FATAL"
		default:
			return severity
		}
	}
	return "INFO"
}

// PinoLevelToString converts pino/bunyan numeric levels to strings.
func PinoLevelToString(level int) string {
	switch level {
	case 10:
		return "TRACE"
	case 20:
		return "DEBUG"
	case 30:
		return "INFO"
	case 40:
		return "WARN"
	case 50:
		return "ERROR"
	case 60:
		return "FATAL"
	default:
		if level < 20 {
			return "TRACE"
		} else if level < 30 {
			return "DEBUG"
		} else if level < 40 {
			return "INFO"
		} else if level < 50 {
			return "WARN"
		} else if level < 60 {
			return "ERROR"
		}
		return "FATAL"
	}
}
