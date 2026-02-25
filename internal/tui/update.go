package tui

import (
	"time"

	"github.com/control-theory/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

type tickDataLoadedMsg struct {
	totalCount      int64
	hasTotalCount   bool
	appList         []string
	hasAppList      bool
	countsHistory   []SeverityCounts
	hasCounts       bool
	words           []model.WordCount
	hasWords        bool
	attributes      []AttributeEntry
	hasAttributes   bool
	logEntries      []model.LogRecord
	hasLogEntries   bool
	drain3Records   []model.LogRecord
	drain3Processed int
	hasDrain3       bool
}

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

		if m.tickInFlight {
			return m, tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
				return TickMsg(t)
			})
		}
		m.tickInFlight = true

		opts := m.queryOpts()
		severityLevels := m.activeSeverityLevels()
		var messagePattern string
		if m.filterRegex != nil {
			messagePattern = m.filterRegex.String()
		}
		logLimit := m.visibleLogLines()
		drainFrom := m.drain3LastProcessed

		// Continue periodic ticks
		return m, tea.Batch(
			m.fetchTickDataCmd(opts, severityLevels, messagePattern, logLimit, drainFrom),
			tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
				return TickMsg(t)
			}),
		)

	case tickDataLoadedMsg:
		m.tickInFlight = false
		m.applyTickData(msg)
		// Visibility-aware refresh: only refresh modal data when it's visible.
		if modal := m.TopModal(); modal != nil {
			if r, ok := modal.(Refreshable); ok {
				r.Refresh()
			}
		}
		return m, nil

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

			// Sidebar rows are mixed pages + apps; resolve click via rendered rows.
			if idx, ok := m.sidebarCursorAtMouseRow(y); ok {
				m.sidebarCursor = idx
				m.activateSidebarCursor()
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

func minuteCountsToSeverity(rows []model.MinuteCounts) []SeverityCounts {
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
	return history
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

func (m *DashboardModel) fetchTickDataCmd(opts model.QueryOpts, severityLevels []string, messagePattern string, logLimit int, drainFrom int) tea.Cmd {
	store := m.store
	if store == nil {
		return func() tea.Msg { return tickDataLoadedMsg{} }
	}

	severityCopy := append([]string(nil), severityLevels...)

	return func() tea.Msg {
		msg := tickDataLoadedMsg{}

		if v, err := store.TotalLogCount(opts); err == nil {
			msg.totalCount = v
			msg.hasTotalCount = true
		}

		if apps, err := store.ListApps(); err == nil {
			msg.appList = apps
			msg.hasAppList = true
		}

		if rows, err := store.SeverityCountsByMinute(opts); err == nil {
			msg.countsHistory = minuteCountsToSeverity(rows)
			msg.hasCounts = true
		}

		if words, err := store.TopWords(20, opts); err == nil {
			msg.words = words
			msg.hasWords = true
		}

		if attrKeys, err := store.TopAttributeKeys(20, opts); err == nil {
			entries := make([]AttributeEntry, len(attrKeys))
			for i, ak := range attrKeys {
				entries[i] = AttributeEntry{
					Key:              ak.Key,
					UniqueValueCount: ak.UniqueValues,
					TotalCount:       ak.TotalCount,
				}
			}
			msg.attributes = entries
			msg.hasAttributes = true
		}

		if msg.hasTotalCount && msg.totalCount > int64(drainFrom) {
			newCount := int(msg.totalCount) - drainFrom
			if newCount > 5000 {
				newCount = 5000
			}
			if newCount > 0 {
				if records, err := store.RecentLogsFiltered(newCount, opts.App, nil, ""); err == nil {
					startIdx := 0
					if len(records) > newCount {
						startIdx = len(records) - newCount
					}
					msg.drain3Records = append([]model.LogRecord(nil), records[startIdx:]...)
					msg.drain3Processed = int(msg.totalCount)
					msg.hasDrain3 = true
				}
			}
		}

		if len(severityCopy) == 0 && severityLevels != nil {
			msg.logEntries = []model.LogRecord{}
			msg.hasLogEntries = true
		} else if records, err := store.RecentLogsFiltered(logLimit, opts.App, severityCopy, messagePattern); err == nil {
			msg.logEntries = records
			msg.hasLogEntries = true
		}

		return msg
	}
}

func (m *DashboardModel) applyTickData(msg tickDataLoadedMsg) {
	if msg.hasTotalCount {
		m.updateProcessingRateStats(msg.totalCount)
	}

	if msg.hasAppList {
		m.appList = msg.appList
		m.clampSidebarCursor()
	}

	if msg.hasCounts {
		m.countsHistory = msg.countsHistory
		m.applyCountsDataToPanels(msg.countsHistory)
	}

	if msg.hasWords {
		m.applyWordsDataToPanels(msg.words)
	}

	if msg.hasAttributes {
		m.applyAttributesDataToPanels(msg.attributes)
	}

	if msg.hasDrain3 {
		m.applyDrain3Records(msg.drain3Records, msg.drain3Processed)
	}

	if msg.hasLogEntries && !m.liveUpdatesPaused() {
		m.applyLogEntries(msg.logEntries)
	}
}

func (m *DashboardModel) applyWordsDataToPanels(words []model.WordCount) {
	if len(m.deckPages) == 0 {
		for _, panel := range m.panels {
			if p, ok := panel.(*WordsChartPanel); ok {
				p.SetData(words)
			}
		}
		return
	}
	for _, page := range m.deckPages {
		for _, panel := range page.Panels {
			if p, ok := panel.(*WordsChartPanel); ok {
				p.SetData(words)
			}
		}
	}
}

func (m *DashboardModel) applyAttributesDataToPanels(entries []AttributeEntry) {
	if len(m.deckPages) == 0 {
		for _, panel := range m.panels {
			if p, ok := panel.(*AttributesChartPanel); ok {
				p.SetData(entries)
			}
		}
		return
	}
	for _, page := range m.deckPages {
		for _, panel := range page.Panels {
			if p, ok := panel.(*AttributesChartPanel); ok {
				p.SetData(entries)
			}
		}
	}
}

func (m *DashboardModel) applyCountsDataToPanels(history []SeverityCounts) {
	if len(m.deckPages) == 0 {
		for _, panel := range m.panels {
			if p, ok := panel.(*CountsChartPanel); ok {
				p.SetData(history)
			}
		}
		return
	}
	for _, page := range m.deckPages {
		for _, panel := range page.Panels {
			if p, ok := panel.(*CountsChartPanel); ok {
				p.SetData(history)
			}
		}
	}
}

func (m *DashboardModel) applyDrain3Records(records []model.LogRecord, processed int) {
	if m.drain3Manager == nil {
		return
	}

	for _, r := range records {
		if r.Message == "" {
			continue
		}
		m.drain3Manager.AddLogMessage(r.Message)
		if drain3Instance, exists := m.drain3BySeverity[r.Level]; exists && drain3Instance != nil {
			drain3Instance.AddLogMessage(r.Message)
		}
	}
	m.drain3LastProcessed = processed
}

func (m *DashboardModel) applyLogEntries(records []model.LogRecord) {
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
