package model

// QueryOpts holds optional filters applied to most queries.
type QueryOpts struct {
	App string // empty = all apps
}

// LogQuerier provides read-only queries on log data.
type LogQuerier interface {
	TotalLogCount(opts QueryOpts) (int64, error)
	TotalLogBytes(opts QueryOpts) (int64, error)
	TopWords(limit int, opts QueryOpts) ([]WordCount, error)
	TopAttributes(limit int, opts QueryOpts) ([]AttributeStat, error)
	TopAttributeKeys(limit int, opts QueryOpts) ([]AttributeKeyStat, error)
	AttributeKeyValues(key string, limit int) (map[string]int64, error)
	SeverityCounts(opts QueryOpts) (map[string]int64, error)
	SeverityCountsByMinute(opts QueryOpts) ([]MinuteCounts, error)
	TopHosts(limit int, opts QueryOpts) ([]DimensionCount, error)
	TopServices(limit int, opts QueryOpts) ([]DimensionCount, error)
	TopServicesBySeverity(severity string, limit int, opts QueryOpts) ([]DimensionCount, error)
	ListApps() ([]string, error)
	RecentLogsFiltered(limit int, app string, severityLevels []string, messagePattern string) ([]LogRecord, error)
}

// SchemaQuerier provides schema introspection and arbitrary read-only queries.
type SchemaQuerier interface {
	ExecuteQuery(query string) ([]map[string]interface{}, error)
	GetSchemaDescription() string
	TableRowCounts() (map[string]int64, error)
}
