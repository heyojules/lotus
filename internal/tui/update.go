package tui

import (
	"time"

	"github.com/control-theory/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages
func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initializeCharts()

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case ActionMsg:
		switch msg.Action {
		case ActionSetSearchTerm:
			if term, ok := msg.Payload.(string); ok {
				m.searchTerm = term
			}
		case ActionPushModal:
			if modal, ok := msg.Payload.(Modal); ok {
				m.PushModal(modal)
			}
		}
		return m, nil

	case tea.MouseMsg:
		return m.handleMouseEvent(msg)

	case ManualResetMsg:
		// Reset drain3 pattern extraction state
		if m.drain3Manager != nil {
			m.drain3Manager.Reset()
			m.drain3LastProcessed = 0
		}
		for _, drain3Instance := range m.drain3BySeverity {
			if drain3Instance != nil {
				drain3Instance.Reset()
			}
		}
		return m, nil

	case TickMsg:
		// Freeze refresh while user is reading logs (or manually paused)
		// so selection/scroll position remains stable.
		if m.liveUpdatesPaused() {
			return m, tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
				return TickMsg(t)
			})
		}

		// Fetch total log count ONCE per tick — shared by stats, drain3, and sidebar
		var totalCount int64
		if m.store != nil {
			if v, err := m.store.TotalLogCount(m.queryOpts()); err == nil {
				totalCount = v
			}
		}

		// Update processing rate statistics using the shared total count
		m.updateProcessingRateStats(totalCount)

		// Refresh app list and per-app counts from DuckDB
		if m.store != nil {
			if apps, err := m.store.ListApps(); err == nil {
				m.appList = apps
			}
			// Cache per-app counts (replaces map entirely each tick)
			counts := make(map[string]int64, len(m.appList))
			for _, app := range m.appList {
				if c, err := m.store.TotalLogCount(model.QueryOpts{App: app}); err == nil {
					counts[app] = c
				}
			}
			m.appCounts = counts

			m.refreshCountsHistoryFromStore()
		}

		// Visibility-aware refresh: only refresh modal data when it's visible.
		if modal := m.TopModal(); modal != nil {
			if r, ok := modal.(Refreshable); ok {
				r.Refresh()
			}
		}

		// Feed drain3 incrementally from DuckDB
		if m.store != nil && m.drain3Manager != nil {
			m.feedDrain3Incremental(totalCount)
		}

		// Refresh chart panel data
		opts := m.queryOpts()
		for _, panel := range m.panels {
			panel.Refresh(m.store, opts)
		}

		m.refreshLogEntriesFromStore()

		// Continue periodic ticks
		return m, tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})

	}

	return m, tea.Batch(cmds...)
}

// handleMouseEvent processes mouse interactions
func (m *DashboardModel) handleMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Modal on stack gets the mouse event first.
	if modal := m.TopModal(); modal != nil {
		pop, cmd := modal.Update(msg)
		if pop {
			m.PopModal()
		}
		return m, cmd
	}

	// Inline handlers (filter/search).
	for _, entry := range m.inlineHandlers {
		if entry.isActive(m) {
			handled, cmd := entry.handler.HandleMouse(m, msg)
			if handled {
				return m, cmd
			}
			break
		}
	}

	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonLeft:
			// Handle left mouse button clicks to switch sections
			return m.handleMouseClick(msg.X, msg.Y)

		case tea.MouseButtonWheelUp:
			// Scroll wheel up = move selection up (like up arrow), or down if reversed
			if m.reverseScrollWheel {
				m.moveSelection(-1)
			} else {
				m.moveSelection(1)
			}
			return m, nil

		case tea.MouseButtonWheelDown:
			// Scroll wheel down = move selection down (like down arrow), or up if reversed
			if m.reverseScrollWheel {
				m.moveSelection(1)
			} else {
				m.moveSelection(-1)
			}
			return m, nil
		}
	}

	return m, nil
}


// handleMouseClick processes mouse clicks to switch between sections
func (m *DashboardModel) handleMouseClick(x, y int) (tea.Model, tea.Cmd) {
	if m.width <= 0 || m.height <= 0 {
		return m, nil
	}

	if m.sidebarVisible {
		if x < sidebarWidth {
			m.activeSection = SectionSidebar

			// Sidebar rows: 0 title, 1 blank, 2 "All", then apps.
			if y >= 2 {
				idx := y - 2
				if idx < 0 {
					idx = 0
				}
				maxIdx := len(m.appList)
				if idx > maxIdx {
					idx = maxIdx
				}
				m.appListIdx = idx

				// Apply app selection on click
				if m.appListIdx == 0 {
					m.selectedApp = "" // "All"
				} else if m.appListIdx-1 < len(m.appList) {
					m.selectedApp = m.appList[m.appListIdx-1]
				}
			}
			return m, nil
		}
		x -= sidebarWidth
	}

	contentWidth := m.width
	if m.sidebarVisible {
		contentWidth -= sidebarWidth
	}

	filterHeight := 0
	if m.hasFilterOrSearch() {
		filterHeight = 1
	}
	usableHeight := m.height - 1 - 2
	chartsHeight := m.calculateRequiredChartsHeight()
	maxChartsHeight := usableHeight - filterHeight - 3
	if maxChartsHeight < 3 {
		maxChartsHeight = 3
	}
	if chartsHeight > maxChartsHeight {
		chartsHeight = maxChartsHeight
	}

	if y < chartsHeight {
		if idx, ok := m.chartPanelAt(contentWidth, chartsHeight, x, y); ok {
			m.activeSection = SectionCharts
			m.activePanelIdx = idx
		}
		return m, nil
	}

	if filterHeight > 0 && y < chartsHeight+filterHeight {
		m.activeSection = SectionFilter
		return m, nil
	}

	m.activeSection = SectionLogs

	return m, nil
}


// refreshCountsHistoryFromStore rebuilds the counts history from DuckDB minute buckets.
func (m *DashboardModel) refreshCountsHistoryFromStore() {
	if m.store == nil {
		return
	}

	rows, err := m.store.SeverityCountsByMinute(m.queryOpts())
	if err != nil {
		return
	}

	history := make([]SeverityCounts, 0, len(rows))
	for _, row := range rows {
		history = append(history, SeverityCounts{
			Trace: int(row.Trace),
			Debug: int(row.Debug),
			Info:  int(row.Info),
			Warn:  int(row.Warn),
			Error: int(row.Error),
			Fatal: int(row.Fatal),
			Total: int(row.Total),
		})
	}
	m.countsHistory = history
}

// activeSeverityLevels returns the list of enabled severity levels when
// severity filtering is active, or nil when all levels are shown.
func (m *DashboardModel) activeSeverityLevels() []string {
	if !m.severityFilterActive {
		return nil
	}
	var levels []string
	for level, enabled := range m.severityFilter {
		if enabled {
			levels = append(levels, level)
		}
	}
	return levels
}

// visibleLogLines returns how many log lines fit on screen given the current
// terminal dimensions, mirroring the layout calculation in renderDashboard.
func (m *DashboardModel) visibleLogLines() int {
	statusLineHeight := 1
	usableHeight := m.height - statusLineHeight - 2
	filterHeight := 0
	if m.hasFilterOrSearch() {
		filterHeight = 1
	}
	chartsHeight := m.calculateRequiredChartsHeight()
	maxChartsHeight := usableHeight - filterHeight - 3
	if maxChartsHeight < 3 {
		maxChartsHeight = 3
	}
	if chartsHeight > maxChartsHeight {
		chartsHeight = maxChartsHeight
	}
	logsHeight := usableHeight - chartsHeight - filterHeight
	if logsHeight < 3 {
		logsHeight = 3
	}
	return logsHeight
}

// feedDrain3Incremental queries DuckDB for logs newer than the last processed count
// and feeds them to drain3 incrementally on each tick.
// totalCount is the pre-fetched TotalLogCount shared across the tick.
func (m *DashboardModel) feedDrain3Incremental(totalCount int64) {
	if totalCount <= int64(m.drain3LastProcessed) {
		return
	}

	// Fetch new logs since last processed
	total := totalCount
	newCount := int(total) - m.drain3LastProcessed
	if newCount > 5000 {
		newCount = 5000 // Cap per-tick to avoid blocking
	}

	records, err := m.store.RecentLogsFiltered(newCount, m.selectedApp, nil, "")
	if err != nil || len(records) == 0 {
		return
	}

	// Feed only the new records (the tail of the result set)
	startIdx := 0
	if len(records) > newCount {
		startIdx = len(records) - newCount
	}

	for i := startIdx; i < len(records); i++ {
		r := records[i]
		if r.Message != "" {
			m.drain3Manager.AddLogMessage(r.Message)
			// Feed severity-specific drain3
			if drain3Instance, exists := m.drain3BySeverity[r.Level]; exists && drain3Instance != nil {
				drain3Instance.AddLogMessage(r.Message)
			}
		}
	}

	m.drain3LastProcessed = int(total)
}

// refreshLogEntriesFromStore queries DuckDB for the filtered log list.
func (m *DashboardModel) refreshLogEntriesFromStore() {
	if m.store == nil {
		return
	}

	severityLevels := m.activeSeverityLevels()
	if m.severityFilterActive && len(severityLevels) == 0 {
		m.logEntries = m.logEntries[:0]
		return
	}

	var messagePattern string
	if m.filterRegex != nil {
		messagePattern = m.filterRegex.String()
	}

	records, err := m.store.RecentLogsFiltered(m.visibleLogLines(), m.selectedApp, severityLevels, messagePattern)
	if err != nil {
		return
	}

	m.logEntries = records

	// Clamp selection to bounds; auto-scroll pins to the latest entry.
	if m.logAutoScroll {
		m.selectedLogIndex = max(0, len(m.logEntries)-1)
	} else if m.selectedLogIndex >= len(m.logEntries) {
		m.selectedLogIndex = max(0, len(m.logEntries)-1)
	}
}


// initializeCharts sets up the charts based on current dimensions
func (m *DashboardModel) initializeCharts() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

}

// updateProcessingRateStats computes processing rate from DuckDB count deltas between ticks.
// totalCount is the pre-fetched TotalLogCount shared across the tick.
func (m *DashboardModel) updateProcessingRateStats(totalCount int64) {
	now := time.Now()

	currentTotal := totalCount
	m.stats.TotalLogsEver = int(currentTotal)

	// Compute delta since last tick
	delta := int(currentTotal) - m.stats.lastTickCount
	if delta < 0 {
		delta = 0
	}

	// Compute elapsed time since last tick
	elapsed := now.Sub(m.stats.lastTickTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	// Compute instantaneous rate (logs per second over this tick interval)
	rate := float64(delta) / elapsed
	if rate > m.stats.PeakLogsPerSec {
		m.stats.PeakLogsPerSec = rate
	}

	// Add to sliding window (one entry per tick)
	m.stats.RecentCounts = append(m.stats.RecentCounts, delta)
	m.stats.RecentTimes = append(m.stats.RecentTimes, now)

	// Keep only entries from the last 10 seconds
	cutoffTime := now.Add(-10 * time.Second)
	for len(m.stats.RecentTimes) > 0 && m.stats.RecentTimes[0].Before(cutoffTime) {
		m.stats.RecentCounts = m.stats.RecentCounts[1:]
		m.stats.RecentTimes = m.stats.RecentTimes[1:]
	}

	// Track for next tick
	m.stats.lastTickCount = int(currentTotal)
	m.stats.lastTickTime = now
	m.stats.LogsThisSecond = delta // Used by formatCurrentRate
}


// getDisplayTimestamp returns the appropriate timestamp based on useLogTime setting
// Falls back to receive time (Timestamp) if OrigTimestamp is not available
func (m *DashboardModel) getDisplayTimestamp(entry model.LogRecord) time.Time {
	if m.useLogTime && !entry.OrigTimestamp.IsZero() {
		return entry.OrigTimestamp
	}
	return entry.Timestamp
}
