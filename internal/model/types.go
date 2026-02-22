package model

import "time"

// LogRecord represents a single log entry used across the system.
// It is the canonical type for storage, transport (socket RPC), and display.
type LogRecord struct {
	Timestamp     time.Time
	OrigTimestamp time.Time // Zero value = no orig timestamp
	Level         string    // TRACE/DEBUG/INFO/WARN/ERROR/FATAL
	LevelNum      int       // Pino numeric: 10/20/30/40/50/60
	Message       string
	RawLine       string
	Service       string
	Hostname      string
	PID           int
	Attributes    map[string]string
	Source        string // "tcp", "stdin"
	App           string // application name, defaults to "default"
}

// WordCount represents a word and its frequency count.
type WordCount struct {
	Word  string
	Count int64
}

// AttributeStat represents an attribute key-value pair and its count.
type AttributeStat struct {
	Key   string
	Value string
	Count int64
}

// AttributeKeyStat represents aggregate stats for an attribute key.
type AttributeKeyStat struct {
	Key          string
	UniqueValues int
	TotalCount   int64
}

// DimensionCount represents grouped counts by a single dimension value
// (for example service or hostname).
type DimensionCount struct {
	Value string
	Count int64
}

// MinuteCounts represents severity counts for one minute.
type MinuteCounts struct {
	Minute time.Time
	Trace  int64
	Debug  int64
	Info   int64
	Warn   int64
	Error  int64
	Fatal  int64
	Total  int64
}
