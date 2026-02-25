package model

import (
	"context"
	"time"
)

// AnalyticsEvent represents a generic analytics/event signal.
// It intentionally mirrors log-style shape for easier pipeline reuse.
type AnalyticsEvent struct {
	Timestamp  time.Time
	Name       string
	Attributes map[string]string
	Source     string
	App        string
}

// MetricSample represents one numeric metric datapoint collected via pull/scrape.
type MetricSample struct {
	Timestamp time.Time
	Name      string
	Value     float64
	Labels    map[string]string
	Source    string
}

// MetricsCollector defines the minimal contract for pull-based metrics collection.
// Implementations can scrape Prometheus/OpenMetrics endpoints on a schedule.
type MetricsCollector interface {
	Name() string
	Collect(ctx context.Context) ([]MetricSample, error)
}
