package ingest

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tinytelemetry/lotus/internal/logparse"
	"github.com/tinytelemetry/lotus/internal/model"
)

// ParseJSONLogEntries parses one JSON line into one or more LogRecords.
// It supports OTEL log data model envelopes and OTEL log-record shape.
func ParseJSONLogEntries(line string) []*model.LogRecord {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil
	}

	records, _ := parseOTELJSONLogEntries(raw, line)
	return records
}

// ParseJSONLogEntry parses a JSON log line into a single LogRecord.
// When an OTEL envelope contains multiple log records, the first entry is returned.
func ParseJSONLogEntry(line string) *model.LogRecord {
	records := ParseJSONLogEntries(line)
	if len(records) == 0 {
		return nil
	}
	return records[0]
}

func parseOTELJSONLogEntries(raw map[string]interface{}, line string) ([]*model.LogRecord, bool) {
	if resourceLogs, ok := raw["resourceLogs"]; ok {
		records := parseOTELResourceLogs(resourceLogs, line)
		return records, true
	}

	if scopeLogs, ok := raw["scopeLogs"]; ok {
		inherited := parseOTELResourceAttributes(raw["resource"])
		records := parseOTELScopeLogs(scopeLogs, inherited, line)
		return records, true
	}

	if instrumentationLogs, ok := raw["instrumentationLibraryLogs"]; ok {
		inherited := parseOTELResourceAttributes(raw["resource"])
		records := parseOTELScopeLogs(instrumentationLogs, inherited, line)
		return records, true
	}

	if logRecords, ok := raw["logRecords"]; ok {
		baseAttrs := parseOTELResourceAttributes(raw["resource"])
		return parseOTELLogRecords(logRecords, baseAttrs, line), true
	}

	if isOTELLogRecord(raw) {
		record := parseOTELLogRecord(raw, nil, line)
		if record == nil {
			return nil, true
		}
		return []*model.LogRecord{record}, true
	}

	return nil, false
}

func parseOTELResourceLogs(value interface{}, line string) []*model.LogRecord {
	resourceLogs, ok := value.([]interface{})
	if !ok {
		return nil
	}

	var records []*model.LogRecord
	for _, item := range resourceLogs {
		resourceLog, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		inherited := parseOTELResourceAttributes(resourceLog["resource"])
		scopeLogsVal := resourceLog["scopeLogs"]
		if scopeLogsVal == nil {
			// Backward compatibility with older OTEL naming.
			scopeLogsVal = resourceLog["instrumentationLibraryLogs"]
		}
		records = append(records, parseOTELScopeLogs(scopeLogsVal, inherited, line)...)
	}
	return records
}

func parseOTELResourceAttributes(value interface{}) map[string]string {
	resource, ok := value.(map[string]interface{})
	if !ok {
		return map[string]string{}
	}
	return parseOTELAttributes(resource["attributes"])
}

func parseOTELScopeLogs(value interface{}, inherited map[string]string, line string) []*model.LogRecord {
	scopeLogs, ok := value.([]interface{})
	if !ok {
		return nil
	}

	var records []*model.LogRecord
	for _, item := range scopeLogs {
		scopeLog, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		scopeAttrs := cloneAttributes(inherited)
		mergeAttributes(scopeAttrs, parseOTELAttributes(scopeLog["attributes"]))

		// OTEL 1.0+ naming.
		if scope, ok := scopeLog["scope"].(map[string]interface{}); ok {
			if name := ExtractStringField(scope, "name"); name != "" {
				scopeAttrs["otel.scope.name"] = name
			}
			if version := ExtractStringField(scope, "version"); version != "" {
				scopeAttrs["otel.scope.version"] = version
			}
			mergeAttributes(scopeAttrs, parseOTELAttributes(scope["attributes"]))
		}

		// Legacy naming.
		if scope, ok := scopeLog["instrumentationLibrary"].(map[string]interface{}); ok {
			if name := ExtractStringField(scope, "name"); name != "" {
				scopeAttrs["otel.scope.name"] = name
			}
			if version := ExtractStringField(scope, "version"); version != "" {
				scopeAttrs["otel.scope.version"] = version
			}
		}

		records = append(records, parseOTELLogRecords(scopeLog["logRecords"], scopeAttrs, line)...)
	}
	return records
}

func parseOTELLogRecords(value interface{}, inherited map[string]string, line string) []*model.LogRecord {
	logRecords, ok := value.([]interface{})
	if !ok {
		return nil
	}

	records := make([]*model.LogRecord, 0, len(logRecords))
	for _, item := range logRecords {
		logRecord, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		record := parseOTELLogRecord(logRecord, inherited, line)
		if record == nil {
			continue
		}
		records = append(records, record)
	}
	return records
}

func parseOTELLogRecord(raw map[string]interface{}, inherited map[string]string, line string) *model.LogRecord {
	receiveTime := time.Now()
	attributes := cloneAttributes(inherited)
	mergeAttributes(attributes, parseOTELAttributes(raw["attributes"]))

	if traceID := ExtractStringField(raw, "traceId"); traceID != "" {
		attributes["trace.id"] = traceID
	}
	if spanID := ExtractStringField(raw, "spanId"); spanID != "" {
		attributes["span.id"] = spanID
	}
	if flags := stringifyJSONValue(raw["flags"]); flags != "" {
		attributes["trace.flags"] = flags
	}
	if dropped := stringifyJSONValue(raw["droppedAttributesCount"]); dropped != "" {
		attributes["otel.dropped_attributes_count"] = dropped
	}

	message := extractOTELBody(raw["body"])

	rawLine := line
	if encoded, err := json.Marshal(raw); err == nil {
		rawLine = string(encoded)
	}
	if message == "" {
		message = rawLine
	}
	message = sanitizeLogMessage(message)

	severityNumber := parseOTELSeverityNumber(raw["severityNumber"])
	severity := ExtractStringField(raw, "severityText")
	if severity == "" && severityNumber > 0 {
		severity = severityFromOTELNumber(severityNumber)
	}
	if severity == "" {
		severity = "INFO"
	}
	normalizedSeverity := logparse.NormalizeSeverity(severity)
	if severityNumber == 0 {
		severityNumber = defaultOTELSeverityNumber(normalizedSeverity)
	}

	origTimestamp := extractOTELTimestamp(raw)

	app := extractAppFromOTELAttributes(attributes)
	if app == "" {
		app = "default"
	}

	return &model.LogRecord{
		Timestamp:     receiveTime,
		OrigTimestamp: origTimestamp,
		Level:         normalizedSeverity,
		LevelNum:      severityNumber,
		Message:       message,
		RawLine:       rawLine,
		Attributes:    attributes,
		App:           app,
	}
}

func parseOTELAttributes(value interface{}) map[string]string {
	out := map[string]string{}
	attributes, ok := value.([]interface{})
	if !ok {
		return out
	}

	for _, item := range attributes {
		attr, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		key := ExtractStringField(attr, "key")
		if key == "" {
			continue
		}
		val := extractOTELAnyValue(attr["value"])
		if val == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func extractOTELBody(value interface{}) string {
	switch body := value.(type) {
	case string:
		return body
	case map[string]interface{}:
		return extractOTELAnyValue(body)
	default:
		return stringifyJSONValue(body)
	}
}

func extractOTELAnyValue(value interface{}) string {
	anyValue, ok := value.(map[string]interface{})
	if !ok {
		return stringifyJSONValue(value)
	}

	for _, key := range []string{"stringValue", "boolValue", "intValue", "doubleValue", "bytesValue"} {
		if val, ok := anyValue[key]; ok {
			return stringifyJSONValue(val)
		}
	}

	if arrayValue, ok := anyValue["arrayValue"].(map[string]interface{}); ok {
		if vals, ok := arrayValue["values"].([]interface{}); ok {
			parts := make([]string, 0, len(vals))
			for _, v := range vals {
				part := extractOTELAnyValue(v)
				if part == "" {
					continue
				}
				parts = append(parts, part)
			}
			return strings.Join(parts, ",")
		}
	}

	if kvListValue, ok := anyValue["kvlistValue"].(map[string]interface{}); ok {
		return stringifyJSONValue(kvListValue["values"])
	}

	return stringifyJSONValue(anyValue)
}

func extractOTELTimestamp(raw map[string]interface{}) time.Time {
	for _, key := range []string{"timeUnixNano", "observedTimeUnixNano"} {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if ts, parsed := parseTimeUnixNano(value); parsed {
			return ts
		}
	}
	return time.Time{}
}

func parseTimeUnixNano(value interface{}) (time.Time, bool) {
	switch v := value.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return time.Time{}, false
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return time.Unix(0, n), true
		}
	case float64:
		return time.Unix(0, int64(v)), true
	case int:
		return time.Unix(0, int64(v)), true
	case int64:
		return time.Unix(0, v), true
	case uint64:
		return time.Unix(0, int64(v)), true
	}
	return time.Time{}, false
}

func parseOTELSeverityNumber(value interface{}) int {
	switch v := value.(type) {
	case float64:
		if v <= 0 {
			return 0
		}
		return int(v)
	case int:
		if v <= 0 {
			return 0
		}
		return v
	case int64:
		if v <= 0 {
			return 0
		}
		return int(v)
	case uint64:
		return int(v)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0
		}
		if n <= 0 {
			return 0
		}
		return n
	default:
		return 0
	}
}

func severityFromOTELNumber(number int) string {
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

func defaultOTELSeverityNumber(level string) int {
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

func isOTELLogRecord(raw map[string]interface{}) bool {
	for _, key := range []string{
		"timeUnixNano",
		"observedTimeUnixNano",
		"severityNumber",
		"severityText",
		"traceId",
		"spanId",
		"flags",
		"droppedAttributesCount",
	} {
		if _, ok := raw[key]; ok {
			return true
		}
	}

	_, hasBody := raw["body"]
	_, hasAttrs := raw["attributes"]
	return hasBody && hasAttrs
}

func extractAppFromOTELAttributes(attributes map[string]string) string {
	for _, key := range []string{"app", "service.name", "service_name", "service", "name"} {
		if value := attributes[key]; value != "" {
			return value
		}
	}
	return ""
}

func cloneAttributes(attributes map[string]string) map[string]string {
	if len(attributes) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(attributes))
	for k, v := range attributes {
		out[k] = v
	}
	return out
}

func mergeAttributes(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

func stringifyJSONValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%v", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	default:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
	}
	return ""
}

func sanitizeLogMessage(message string) string {
	clean := strings.ReplaceAll(message, "\t", " ")
	clean = strings.ReplaceAll(clean, "\n", " ")
	clean = strings.ReplaceAll(clean, "\r", " ")
	return clean
}

// ExtractStringField returns the first non-empty string value found among the given keys.
func ExtractStringField(raw map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := raw[k]; ok {
			if str := stringifyJSONValue(v); str != "" {
				return str
			}
		}
	}
	return ""
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
