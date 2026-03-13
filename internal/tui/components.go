package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderBranding renders "Tiny Telemetry!" with a green to light blue gradient
func (m *DashboardModel) renderBranding() string {
	colors := []string{
		"#49E209", // T
		"#44E017", // i
		"#3FDF24", // n
		"#39DD32", // y
		"#34DB3F", // (space)
		"#2FDA4D", // T
		"#2AD85A", // e
		"#24D668", // l
		"#1FD575", // e
		"#1AD383", // m
		"#15D190", // e
		"#10D09E", // t
		"#0ACEAB", // r
		"#05CCB9", // y
		"#00CAC7", // !
	}

	chars := []string{"T", "i", "n", "y", " ", "T", "e", "l", "e", "m", "e", "t", "r", "y", "!"}

	var result string
	for i, char := range chars {
		style := lipgloss.NewStyle().
			Background(ColorNavy).
			Foreground(lipgloss.Color(colors[i])).Bold(true)
		result += style.Render(char)
	}

	return result
}

// renderStatusLine renders the status/help line at the bottom of the screen
func (m *DashboardModel) renderStatusLine() string {
	// Create base style for the status line
	baseStyle := lipgloss.NewStyle().
		Background(ColorNavy).
		Foreground(ColorWhite)

	var statusText string
	var leftText string
	var rightText string

	// Use content width (accounting for sidebar) for all width calculations
	w := m.contentWidth()

	// Determine available width categories
	veryNarrow := w < 60
	narrow := w < 80
	medium := w < 120

	// Build left section: view tabs for the active page
	if !m.filterActive && !m.searchActive {
		pg := m.activePage()
		if pg != nil && len(pg.Views) > 1 && !veryNarrow {
			var tabs []string
			for i, vw := range pg.Views {
				if i == pg.ActiveViewIdx {
					tab := lipgloss.NewStyle().
						Background(ColorNavy).
						Foreground(ColorBlue).
						Bold(true).
						Render(vw.Title)
					tabs = append(tabs, tab)
				} else {
					tab := lipgloss.NewStyle().
						Background(ColorNavy).
						Foreground(ColorGray).
						Render(vw.Title)
					tabs = append(tabs, tab)
				}
			}
			leftText = strings.Join(tabs, baseStyle.Render(" "))
		} else if pg != nil && len(pg.Views) == 1 {
			leftText = baseStyle.Render(pg.Views[0].Title)
		}
	}

	// Build center section (status/help text) - dynamically adjust based on width
	if m.filterActive {
		if narrow {
			statusText = "Enter: Apply • ESC: Cancel"
		} else {
			statusText = "Type regex pattern • Enter: Apply • ESC: Cancel"
		}
	} else if m.searchActive {
		if narrow {
			statusText = "Enter: Apply • ESC: Cancel"
		} else {
			statusText = "Type search term • Enter: Apply • ESC: Cancel"
		}
	} else if m.activeSection == SectionLogs {
		if veryNarrow {
			statusText = "?: Help • ↑↓ Nav • Enter"
		} else if narrow {
			statusText = "?: Help • ↑↓ Navigate • Enter: Details"
		} else if medium {
			statusText = "?: Help • ↑↓: Navigate • Home/End • PgUp/Dn • Enter: Details • []: View"
		} else {
			statusText = "?: Help • Wheel: scroll • ↑↓: Navigate • Home: Top • End: Latest • PgUp/PgDn: Page • []: Switch view • Enter: Details"
		}
	} else if m.HasModal() {
		statusText = "ESC: Close"
	} else {
		// Default status showing main actions
		if veryNarrow {
			statusText = "Tab • Space • i • ? • q"
		} else if narrow {
			statusText = "?: Help • Tab: Nav • []: View • Space: Pause • q: Quit"
		} else if medium {
			statusText = "Tab: Navigate • []: Switch View • Space: Pause • i: Stats • Enter: Select • q: Quit"
		} else {
			statusText = "?: Help • Click sections • Wheel: scroll • []: Switch view • Space: Pause • Tab: Navigate • i: Stats • Enter: Select • q: Quit"
		}
	}

	// Build right section (status info and branding)
	var statusInfo string

	// Check for version updates
	var versionUpdateInfo string
	if m.versionInfo != nil && m.versionInfo.HasUpdate {
		versionUpdateInfo = fmt.Sprintf("🔄 v%s available", m.versionInfo.LatestVersion)
	}

	if !m.filterActive && !m.searchActive && !m.HasModal() {
		if m.liveUpdatesPaused() {
			if m.viewPaused {
				statusInfo = "⏸ Manual"
			} else {
				statusInfo = "⏸ Focus Lock"
			}
		} else if !veryNarrow {
			intervalStr := m.formatDuration(m.updateInterval)
			if narrow {
				statusInfo = intervalStr
			} else {
				statusInfo = fmt.Sprintf("Update: %s", intervalStr)
			}
		}
	}

	// Add data source connectivity indicator
	var dataSourceInfo string
	if m.dataSource != "" && !veryNarrow {
		var dot string
		stale := time.Since(m.lastTickAt) > 3*m.updateInterval
		if !m.lastTickOK {
			dot = lipgloss.NewStyle().Background(ColorNavy).Foreground(lipgloss.Color("#FF4444")).Render("●")
		} else if stale {
			dot = lipgloss.NewStyle().Background(ColorNavy).Foreground(lipgloss.Color("#FFAA00")).Render("●")
		} else {
			dot = lipgloss.NewStyle().Background(ColorNavy).Foreground(lipgloss.Color("#44FF44")).Render("●")
		}
		dataSourceInfo = dot + " " + m.dataSource
	}

	// Add timestamp mode indicator
	var timestampMode string
	if m.useLogTime {
		if narrow {
			timestampMode = "⏱Log"
		} else {
			timestampMode = "⏱ Log Time"
		}
	}

	// Add DB error indicator (auto-clears after 30s)
	var dbErrorInfo string
	if m.lastError != "" && time.Since(m.lastErrorAt) < 30*time.Second {
		dbErrorStyle := lipgloss.NewStyle().
			Background(ColorNavy).
			Foreground(lipgloss.Color("#FF6666")).
			Faint(true)
		dbErrorInfo = dbErrorStyle.Render("DB error")
	}

	// Combine status info, timestamp mode, and version update
	var rightParts []string
	if dbErrorInfo != "" {
		rightParts = append(rightParts, dbErrorInfo)
	}
	if dataSourceInfo != "" {
		rightParts = append(rightParts, dataSourceInfo)
	}
	if statusInfo != "" {
		rightParts = append(rightParts, statusInfo)
	}
	if timestampMode != "" {
		rightParts = append(rightParts, timestampMode)
	}
	if versionUpdateInfo != "" {
		rightParts = append(rightParts, versionUpdateInfo)
	}
	if w >= 30 {
		rightParts = append(rightParts, m.brandingCache)
	}

	if len(rightParts) > 0 {
		rightText = strings.Join(rightParts, "  ")
	}

	// Calculate dynamic widths based on available space using visible width
	leftWidth := lipgloss.Width(leftText) + 2   // Add some padding
	rightWidth := lipgloss.Width(rightText) + 2 // Add some padding

	// Ensure minimum widths don't exceed available width
	if leftWidth+rightWidth >= w {
		// Too narrow, just show what fits
		if w < 20 {
			// Extremely narrow - just show section name
			return baseStyle.Width(w).Render(leftText)
		}
		// Show abbreviated content
		leftWidth = min(10, w/3)
		rightWidth = min(15, w/3)
	}

	// Calculate center width (remaining space)
	centerWidth := w - leftWidth - rightWidth
	if centerWidth < 0 {
		centerWidth = 0
	}

	// Apply styles with calculated widths
	leftStyle := baseStyle.Align(lipgloss.Left).Width(leftWidth)
	centerStyle := baseStyle.Align(lipgloss.Center).Width(centerWidth)
	rightStyle := baseStyle.Align(lipgloss.Right).Width(rightWidth)

	// Truncate content if necessary to prevent wrapping
	if lipgloss.Width(leftText) > leftWidth {
		leftText = leftText[:max(0, leftWidth-1)]
	}
	if lipgloss.Width(statusText) > centerWidth {
		statusText = statusText[:max(0, centerWidth-1)]
	}
	if lipgloss.Width(rightText) > rightWidth {
		// Don't truncate styled text as it would break ANSI codes
		// Instead, only show what fits based on priority
		if statusInfo != "" && w < 50 {
			rightText = statusInfo // Drop branding if too narrow
		} else if w < 40 {
			rightText = "" // Drop everything if extremely narrow
		}
	}

	leftPart := leftStyle.Render(leftText)
	centerPart := centerStyle.Render(statusText)
	rightPart := rightStyle.Render(rightText)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPart, centerPart, rightPart)
}

// renderFilter renders the filter or search input section
func (m *DashboardModel) renderFilter() string {
	var title, content string
	var styleColor lipgloss.Color

	// Check what to display based on active state and applied filters/searches
	if m.filterActive {
		// Actively editing filter
		title = "🔍 Filter (editing)"
		content = m.filterInput.View()
		styleColor = ColorGreen
		if m.filterRegex != nil {
			content += fmt.Sprintf(" | Showing: %d/%d entries", len(m.logEntries), m.currentTotalLogs())
		}
	} else if m.searchActive {
		// Actively editing search
		title = "🔎 Search (editing)"
		content = m.searchInput.View()
		styleColor = ColorYellow
		if m.searchTerm != "" {
			content += fmt.Sprintf(" | Highlighting: %q", m.searchTerm)
		}
	} else if m.filterRegex != nil || m.filterInput.Value() != "" {
		// Filter applied but not editing - show the filter value
		title = "🔍 Filter"
		content = fmt.Sprintf("[%s]", m.filterInput.Value())
		styleColor = ColorGreen
		content += fmt.Sprintf(" | Showing: %d/%d entries", len(m.logEntries), m.currentTotalLogs())
		content += " | Press '/' to edit"
	} else if m.searchTerm != "" || m.searchInput.Value() != "" {
		// Search applied but not editing - show the search term
		title = "🔎 Search"
		searchValue := m.searchTerm
		if searchValue == "" {
			searchValue = m.searchInput.Value()
		}
		content = fmt.Sprintf("[%s]", searchValue)
		styleColor = ColorYellow
		content += fmt.Sprintf(" | Highlighting: %q", searchValue)
		content += " | Press 's' to edit"
	} else {
		// Nothing active or applied
		return ""
	}

	// Minimal style without borders for filter/search
	minimalFilterStyle := lipgloss.NewStyle().
		Foreground(styleColor).
		Padding(0, 1)

	return minimalFilterStyle.Render(title + " " + content)
}

// clampInstructionsScroll clamps the instructions scroll offset to valid bounds.
// Must be called from Update (after applyTickData and window resize) to keep
// View pure / side-effect-free.
func (m *DashboardModel) clampInstructionsScroll(availableLines int) {
	if availableLines < 1 {
		availableLines = 1
	}
	// The instructions list length is fixed; use the same constant set that
	// renderLogScrollContent builds so the max-scroll calculation stays in sync.
	const instructionsLen = 15 // base instruction lines (see renderLogScrollContent)
	maxScroll := instructionsLen - availableLines + 1
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.instructionsScrollOffset > maxScroll {
		m.instructionsScrollOffset = maxScroll
	}
	if m.instructionsScrollOffset < 0 {
		m.instructionsScrollOffset = 0
	}
}

// renderLogScrollContent generates the log content without border wrapper
func (m *DashboardModel) renderLogScrollContent(height int, logWidth int) []string {
	var logLines []string

	// Add focus-lock indicator and help text when log section is active.
	if m.activeSection == SectionLogs {
		pausedStyle := lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)
		statusLine := pausedStyle.Render("Focus lock on: live updates paused while reading logs • Tab/click away to resume")
		logLines = append(logLines, statusLine)
		height-- // Reduce available height for logs
	}

	// Add column headers when columns are enabled
	if m.showColumns {
		timestampHeader := lipgloss.NewStyle().Foreground(ColorWhite).Render("Time    ")
		severityHeader := lipgloss.NewStyle().Foreground(ColorWhite).Render("Level")

		// Use k8s headers if recent logs have k8s attributes, otherwise use host/service headers
		var col1Header, col2Header string
		if m.hasK8sAttributes() {
			col1Header = lipgloss.NewStyle().Foreground(ColorWhite).Render("Namespace           ")
			col2Header = lipgloss.NewStyle().Foreground(ColorWhite).Render("Pod                 ")
		} else {
			col1Header = lipgloss.NewStyle().Foreground(ColorWhite).Render("Host        ")
			col2Header = lipgloss.NewStyle().Foreground(ColorWhite).Render("Service         ")
		}
		messageHeader := lipgloss.NewStyle().Foreground(ColorWhite).Render("Message")

		headerLine := fmt.Sprintf("%s %s %s %s %s",
			timestampHeader, severityHeader, col1Header, col2Header, messageHeader)
		logLines = append(logLines, headerLine)
		height-- // Reduce available height for logs
	}

	// Show recent log entries
	startIdx := 0
	maxLines := height // Use all remaining space after accounting for paused status and headers
	if maxLines < 1 {
		maxLines = 1
	}

	// When in log section or log viewer modal, don't auto-scroll to latest
	if m.activeSection != SectionLogs && !m.isLogViewerOpen() && len(m.logEntries) > maxLines {
		startIdx = len(m.logEntries) - maxLines
	} else if m.activeSection == SectionLogs || m.isLogViewerOpen() {
		// Keep selected log in view
		if m.selectedLogIndex >= 0 && m.selectedLogIndex < len(m.logEntries) {
			// Center selected log if possible
			startIdx = m.selectedLogIndex - maxLines/2
			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx+maxLines > len(m.logEntries) {
				startIdx = max(0, len(m.logEntries)-maxLines)
			}
		}
	}

	for i := startIdx; i < len(m.logEntries) && i < startIdx+maxLines; i++ {
		entry := m.logEntries[i]
		isSelected := (m.activeSection == SectionLogs || m.isLogViewerOpen()) && i == m.selectedLogIndex
		formatted := m.formatLogEntry(entry, logWidth, isSelected)
		logLines = append(logLines, formatted)
	}

	if len(logLines) <= 1 { // Only status line
		// Add helpful instructions when no logs are available
		instructions := []string{
			"Waiting for log entries...",
			"",
			"💡 To get started:",
			"  • Pipe logs: cat mylog.json | tiny-telemetry",
			"  • Stream logs: kubectl logs -f pod | tiny-telemetry",
			"  • From file: tiny-telemetry -f application.log -f other.log -f 'dir/*.globlog'",
			"",
		}

		// Add current filters section if any are applied
		filterStatus := m.buildFilterStatus()
		if len(filterStatus) > 0 {
			instructions = append(instructions, "🔍 Current filters:")
			instructions = append(instructions, filterStatus...)
			instructions = append(instructions, "")
		}

		instructions = append(instructions, []string{
			"📋 Key commands:",
			"  • ?/h: Show help",
			"  • /: Filter logs (message & attributes)",
			"  • Ctrl+f: Filter logs by severity",
			"  • s: Search and highlight",
			"  • Tab: Navigate sections",
			"  • q: Quit",
		}...)

		// Handle scrolling for instructions if they exceed available height
		availableLines := height - 1 // Reserve space for status line that's already added
		if availableLines < 1 {
			availableLines = 1
		}

		if len(instructions) > availableLines {
			// Scroll offset is already clamped in Update via clampInstructionsScroll.
			scrollOffset := m.instructionsScrollOffset

			// Add scroll up indicator if not at top
			if scrollOffset > 0 {
				scrollUpIndicator := lipgloss.NewStyle().
					Foreground(ColorGray).
					Render(fmt.Sprintf("  ↑ %d more lines above", scrollOffset))
				logLines = append(logLines, scrollUpIndicator)
				availableLines-- // Use one line for indicator
			}

			// Show visible portion of instructions
			endIdx := scrollOffset + availableLines
			if endIdx > len(instructions) {
				endIdx = len(instructions)
			}

			// Reserve space for bottom scroll indicator if needed
			if endIdx < len(instructions) {
				availableLines-- // Reserve space for bottom indicator
				endIdx = scrollOffset + availableLines
			}

			// Add visible instructions
			visibleInstructions := instructions[scrollOffset:endIdx]
			logLines = append(logLines, visibleInstructions...)

			// Add scroll down indicator if not at bottom
			if endIdx < len(instructions) {
				remaining := len(instructions) - endIdx
				scrollDownIndicator := lipgloss.NewStyle().
					Foreground(ColorGray).
					Render(fmt.Sprintf("  ↓ %d more lines below (use ↑↓ or k/j to scroll)", remaining))
				logLines = append(logLines, scrollDownIndicator)
			}
		} else {
			// All instructions fit, no scrolling needed
			logLines = append(logLines, instructions...)
		}
	}

	return logLines
}

// buildFilterStatus returns a list of currently applied filters for display when no logs are shown
func (m *DashboardModel) buildFilterStatus() []string {
	var filters []string

	// Check severity filter
	if m.severityFilterActive {
		disabledSeverities := []string{}
		enabledSeverities := []string{}

		severityLevels := []string{"FATAL", "CRITICAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE", "UNKNOWN"}
		for _, severity := range severityLevels {
			if enabled, exists := m.severityFilter[severity]; exists {
				if enabled {
					enabledSeverities = append(enabledSeverities, severity)
				} else {
					disabledSeverities = append(disabledSeverities, severity)
				}
			}
		}

		if len(enabledSeverities) > 0 && len(enabledSeverities) < len(severityLevels) {
			if len(enabledSeverities) <= 3 {
				filters = append(filters, "  • Severity: Only showing "+joinWithCommas(enabledSeverities))
			} else {
				filters = append(filters, "  • Severity: Hiding "+joinWithCommas(disabledSeverities))
			}
		} else if len(enabledSeverities) == 0 {
			filters = append(filters, "  • Severity: All severities disabled (no logs will show)")
		}
	}

	// Check regex filter
	if m.filterRegex != nil {
		pattern := m.filterInput.Value()
		if pattern == "" && m.filterRegex != nil {
			pattern = m.filterRegex.String()
		}
		if pattern != "" {
			filters = append(filters, "  • Regex filter: "+pattern)
		}
	}

	// Check search term
	if m.searchTerm != "" {
		filters = append(filters, "  • Search highlight: "+m.searchTerm)
	}

	// Add instructions for clearing filters if any are active
	if len(filters) > 0 {
		filters = append(filters, "")
		filters = append(filters, "  💡 To clear filters:")
		if m.severityFilterActive {
			filters = append(filters, "    • Ctrl+F → Select All → Enter (enable all severities)")
		}
		if m.filterRegex != nil {
			filters = append(filters, "    • / → Backspace/Delete → Enter (clear regex)")
		}
		if m.searchTerm != "" {
			filters = append(filters, "    • s → Backspace/Delete → Enter (clear search)")
		}
	}

	return filters
}

// joinWithCommas joins a slice of strings with commas and "and" before the last item
func joinWithCommas(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}

	result := ""
	for i, item := range items {
		if i == len(items)-1 {
			result += "and " + item
		} else {
			result += item + ", "
		}
	}
	return result
}

// renderLogScroll renders the scrolling log section
func (m *DashboardModel) renderLogScroll(height int) string {
	// Use content width (sidebar-adjusted) for logs
	logWidth := m.contentWidth() - 2 // Account for borders
	if logWidth < 40 {
		logWidth = 40 // Higher minimum for readability
	}

	// Highlight border when log section is active
	borderColor := ColorNavy
	if m.activeSection == SectionLogs {
		borderColor = ColorBlue
	}

	style := sectionStyle.
		Width(logWidth).
		Height(height).
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor)

	// Get log content
	logLines := m.renderLogScrollContent(height, logWidth)

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, logLines...))
}
