package ingest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/control-theory/lotus/internal/logparse"
	"github.com/control-theory/lotus/internal/model"
	"github.com/control-theory/lotus/internal/timestamp"
)

// shared timestamp parser for JSON timestamp extraction
var tsParser = timestamp.NewParser()

// ParseJSONLogEntry parses a JSON log line into a LogRecord.
// Supports common JSON log formats: pino, bunyan, winston, zerolog, zap, logrus, etc.
func ParseJSONLogEntry(line string) *model.LogRecord {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil
	}

	receiveTime := time.Now()
	record := &model.LogRecord{
		Timestamp:  receiveTime,
		RawLine:    line,
		Attributes: make(map[string]string),
	}

	// Extract message (try common field names)
	record.Message = ExtractStringField(raw, "msg", "message", "body", "text", "log")
	if record.Message == "" {
		// Fallback: use the whole JSON as message
		record.Message = line
	}
	// Clean message
	record.Message = strings.ReplaceAll(record.Message, "\t", " ")
	record.Message = strings.ReplaceAll(record.Message, "\n", " ")
	record.Message = strings.ReplaceAll(record.Message, "\r", " ")

	// Extract severity/level
	severity := ExtractLevelFromJSON(raw)
	record.Level = logparse.NormalizeSeverity(severity)

	// Extract original timestamp
	record.OrigTimestamp = ExtractTimestampFromJSON(raw)

	// Extract application name
	app := ExtractStringField(raw, "_app")
	if app == "" {
		app = "default"
	}
	record.App = app

	// Build attributes from remaining fields
	// Known fields to skip (already extracted above)
	skip := map[string]bool{
		"msg": true, "message": true, "body": true, "text": true, "log": true,
		"level": true, "severity": true, "levelname": true, "loglevel": true, "lvl": true,
		"time": true, "timestamp": true, "ts": true, "t": true, "@timestamp": true, "date": true, "datetime": true,
		"_app": true,
	}
	for k, v := range raw {
		if skip[k] {
			continue
		}
		switch val := v.(type) {
		case string:
			record.Attributes[k] = val
		case float64:
			record.Attributes[k] = fmt.Sprintf("%v", val)
		case bool:
			record.Attributes[k] = fmt.Sprintf("%v", val)
		default:
			if b, err := json.Marshal(val); err == nil {
				record.Attributes[k] = string(b)
			}
		}
	}

	return record
}

// ExtractStringField returns the first non-empty string value found among the given keys.
func ExtractStringField(raw map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := raw[k]; ok {
			switch val := v.(type) {
			case string:
				return val
			case float64:
				return fmt.Sprintf("%v", val)
			}
		}
	}
	return ""
}

// ExtractLevelFromJSON extracts the log level from common JSON field names.
func ExtractLevelFromJSON(raw map[string]interface{}) string {
	// Try string-based level fields
	for _, key := range []string{"level", "severity", "levelname", "loglevel", "lvl"} {
		if v, ok := raw[key]; ok {
			switch val := v.(type) {
			case string:
				return val
			case float64:
				// Pino/bunyan numeric levels
				return logparse.PinoLevelToString(int(val))
			}
		}
	}
	return "INFO"
}

// ExtractTimestampFromJSON extracts the original log timestamp from common JSON fields.
// Delegates to timestamp.Parser for format detection and parsing.
func ExtractTimestampFromJSON(raw map[string]interface{}) time.Time {
	for _, key := range []string{"time", "timestamp", "ts", "t", "@timestamp", "date", "datetime"} {
		v, ok := raw[key]
		if !ok {
			continue
		}
		if t, found := tsParser.ParseTimestamp(v); found {
			return t
		}
	}
	return time.Time{} // zero = no original timestamp
}

// CreateFallbackLogEntry creates a basic LogRecord for unparseable lines.
func CreateFallbackLogEntry(line string) *model.LogRecord {
	// Replace tabs and newlines with spaces to prevent formatting issues
	cleanLine := strings.ReplaceAll(line, "\t", " ")
	cleanLine = strings.ReplaceAll(cleanLine, "\n", " ")
	cleanLine = strings.ReplaceAll(cleanLine, "\r", " ")

	severity := logparse.ExtractSeverityFromText(cleanLine)
	receiveTime := time.Now()

	return &model.LogRecord{
		Timestamp:     receiveTime,
		OrigTimestamp: time.Time{},
		Level:         logparse.NormalizeSeverity(severity),
		Message:       cleanLine,
		RawLine:       cleanLine,
		Attributes:    make(map[string]string),
		App:           "default",
	}
}

// ExtractService extracts the service name from log attributes.
func ExtractService(attributes map[string]string) string {
	if s := attributes["service.name"]; s != "" {
		return s
	}
	if s := attributes["service"]; s != "" {
		return s
	}
	if s := attributes["serviceName"]; s != "" {
		return s
	}
	if s := attributes["app"]; s != "" {
		return s
	}
	if s := attributes["name"]; s != "" {
		return s
	}
	return "unknown"
}
