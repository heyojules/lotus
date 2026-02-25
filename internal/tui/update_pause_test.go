package tui

import (
	"testing"
	"time"

	"github.com/tinytelemetry/lotus/internal/model"
)

type countingStore struct {
	totalCount int64

	totalLogCountCalls        int
	totalLogBytesCalls        int
	topWordsCalls             int
	topAttributesCalls        int
	topAttributeKeysCalls     int
	attributeKeyValuesCalls   int
	severityCountsCalls       int
	severityByMinuteCalls     int
	topHostsCalls             int
	topServicesCalls          int
	topServicesBySeverityCall int
	listAppsCalls             int
	recentLogsFilteredCalls   int

	recentLogs []model.LogRecord
}

func (s *countingStore) TotalLogCount(_ model.QueryOpts) (int64, error) {
	s.totalLogCountCalls++
	return s.totalCount, nil
}

func (s *countingStore) TotalLogBytes(_ model.QueryOpts) (int64, error) {
	s.totalLogBytesCalls++
	return 0, nil
}

func (s *countingStore) TopWords(_ int, _ model.QueryOpts) ([]model.WordCount, error) {
	s.topWordsCalls++
	return []model.WordCount{}, nil
}

func (s *countingStore) TopAttributes(_ int, _ model.QueryOpts) ([]model.AttributeStat, error) {
	s.topAttributesCalls++
	return []model.AttributeStat{}, nil
}

func (s *countingStore) TopAttributeKeys(_ int, _ model.QueryOpts) ([]model.AttributeKeyStat, error) {
	s.topAttributeKeysCalls++
	return []model.AttributeKeyStat{}, nil
}

func (s *countingStore) AttributeKeyValues(_ string, _ int) (map[string]int64, error) {
	s.attributeKeyValuesCalls++
	return map[string]int64{}, nil
}

func (s *countingStore) SeverityCounts(_ model.QueryOpts) (map[string]int64, error) {
	s.severityCountsCalls++
	return map[string]int64{}, nil
}

func (s *countingStore) SeverityCountsByMinute(_ model.QueryOpts) ([]model.MinuteCounts, error) {
	s.severityByMinuteCalls++
	return []model.MinuteCounts{}, nil
}

func (s *countingStore) TopHosts(_ int, _ model.QueryOpts) ([]model.DimensionCount, error) {
	s.topHostsCalls++
	return []model.DimensionCount{}, nil
}

func (s *countingStore) TopServices(_ int, _ model.QueryOpts) ([]model.DimensionCount, error) {
	s.topServicesCalls++
	return []model.DimensionCount{}, nil
}

func (s *countingStore) TopServicesBySeverity(_ string, _ int, _ model.QueryOpts) ([]model.DimensionCount, error) {
	s.topServicesBySeverityCall++
	return []model.DimensionCount{}, nil
}

func (s *countingStore) ListApps() ([]string, error) {
	s.listAppsCalls++
	return []string{}, nil
}

func (s *countingStore) RecentLogsFiltered(_ int, _ string, _ []string, _ string) ([]model.LogRecord, error) {
	s.recentLogsFilteredCalls++
	return s.recentLogs, nil
}

func TestTick_AutoPausesWhenLogsFocused(t *testing.T) {
	t.Parallel()

	store := &countingStore{
		totalCount: 5,
		recentLogs: []model.LogRecord{
			{Message: "new entry", Level: "INFO", Timestamp: time.Now()},
		},
	}

	m := NewDashboardModel(1000, time.Second, false, false, store, "")
	m.activeSection = SectionLogs
	m.logEntries = []model.LogRecord{
		{Message: "pinned", Level: "INFO", Timestamp: time.Now().Add(-time.Minute)},
	}

	m.Update(TickMsg(time.Now()))

	if store.totalLogCountCalls != 0 {
		t.Fatalf("total log count calls = %d, want 0 while logs focused", store.totalLogCountCalls)
	}
	if store.recentLogsFilteredCalls != 0 {
		t.Fatalf("recent logs calls = %d, want 0 while logs focused", store.recentLogsFilteredCalls)
	}
	if got := len(m.logEntries); got != 1 || m.logEntries[0].Message != "pinned" {
		t.Fatalf("log entries changed while focused: got %+v", m.logEntries)
	}
}

func TestTick_ManualPauseSkipsRefresh(t *testing.T) {
	t.Parallel()

	store := &countingStore{
		totalCount: 3,
		recentLogs: []model.LogRecord{
			{Message: "should-not-load", Level: "INFO", Timestamp: time.Now()},
		},
	}

	m := NewDashboardModel(1000, time.Second, false, false, store, "")
	m.viewPaused = true
	m.activeSection = SectionCharts

	m.Update(TickMsg(time.Now()))

	if store.totalLogCountCalls != 0 {
		t.Fatalf("total log count calls = %d, want 0 while manually paused", store.totalLogCountCalls)
	}
	if store.recentLogsFilteredCalls != 0 {
		t.Fatalf("recent logs calls = %d, want 0 while manually paused", store.recentLogsFilteredCalls)
	}
}

func TestTick_ResumesAfterLeavingLogs(t *testing.T) {
	t.Parallel()

	store := &countingStore{
		totalCount: 2,
		recentLogs: []model.LogRecord{
			{Message: "fresh", Level: "INFO", Timestamp: time.Now()},
		},
	}

	m := NewDashboardModel(1000, time.Second, false, false, store, "")
	m.activeSection = SectionLogs
	m.logEntries = []model.LogRecord{
		{Message: "old", Level: "INFO", Timestamp: time.Now().Add(-time.Minute)},
	}

	m.Update(TickMsg(time.Now()))
	if got := m.logEntries[0].Message; got != "old" {
		t.Fatalf("log message changed while focused: got %q", got)
	}

	m.activeSection = SectionCharts
	m.Update(TickMsg(time.Now()))
	if !m.tickInFlight {
		t.Fatal("expected async tick fetch to be in-flight after leaving logs")
	}

	var messagePattern string
	if m.filterRegex != nil {
		messagePattern = m.filterRegex.String()
	}
	msg := m.fetchTickDataCmd(
		m.queryOpts(),
		m.activeSeverityLevels(),
		messagePattern,
		m.visibleLogLines(),
		m.drain3LastProcessed,
	)()
	m.Update(msg)

	if store.totalLogCountCalls == 0 {
		t.Fatal("expected refresh calls after leaving logs, got none")
	}
	if store.recentLogsFilteredCalls == 0 {
		t.Fatal("expected log refresh after leaving logs, got none")
	}
	if got := len(m.logEntries); got != 1 || m.logEntries[0].Message != "fresh" {
		t.Fatalf("log entries not refreshed after leaving logs: got %+v", m.logEntries)
	}
}
