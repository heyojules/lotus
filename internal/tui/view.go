package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// contentWidth returns the width available for main content, accounting for sidebar.
func (m *DashboardModel) contentWidth() int {
	if m.sidebarVisible {
		w := m.width - sidebarWidth
		if w < 40 {
			w = 40
		}
		return w
	}
	return m.width
}

// hasFilterOrSearch returns true if a filter or search is active or applied
func (m *DashboardModel) hasFilterOrSearch() bool {
	return m.filterActive || m.searchActive ||
		m.filterRegex != nil || m.filterInput.Value() != "" ||
		m.searchTerm != "" || m.searchInput.Value() != ""
}

// View renders the dashboard
func (m *DashboardModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Initializing dashboard..."
	}

	// If a modal is on the stack, render it full-screen.
	if modal := m.TopModal(); modal != nil {
		return modal.View(m.width, m.height)
	}

	// Main dashboard layout
	return m.renderDashboard()
}

// renderDashboard renders the main dashboard layout
func (m *DashboardModel) renderDashboard() string {
	// Ensure minimum height
	if m.height < 20 {
		return "Terminal too small. Resize to at least 20 lines."
	}

	// Determine content width (accounting for sidebar)
	contentWidth := m.contentWidth()
	showSidebar := m.sidebarVisible

	// Calculate required space for charts dynamically
	requiredChartsHeight := m.calculateRequiredChartsHeight()

	// Filter/Search height depends on whether filter or search is applied (or being edited)
	filterHeight := 0 // No space when inactive
	if m.hasFilterOrSearch() {
		filterHeight = 1 // Single row for filter/search
	}

	// Reserve space for status line at bottom
	statusLineHeight := 1

	// Use full height for proper layout.
	usableHeight := m.height - statusLineHeight - 2
	minLogsHeight := 3
	maxChartsHeight := usableHeight - filterHeight - minLogsHeight
	if maxChartsHeight < 3 {
		maxChartsHeight = 3
	}

	chartsHeight := requiredChartsHeight
	if chartsHeight > maxChartsHeight {
		chartsHeight = maxChartsHeight
	}

	logsHeight := usableHeight - chartsHeight - filterHeight
	if logsHeight < minLogsHeight {
		logsHeight = minLogsHeight
	}

	// Top section: dynamic chart grid.
	topSection := m.renderChartsGrid(contentWidth, chartsHeight)

	// Middle section: Filter (only when active)
	var sections []string
	sections = append(sections, topSection)

	if m.hasFilterOrSearch() {
		filterSection := m.renderFilter()
		sections = append(sections, filterSection)
	}

	// Bottom section: Log scroll
	logsSection := m.renderLogScroll(logsHeight)
	sections = append(sections, logsSection)

	// Combine sections with strict height constraints
	mainContent := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Add status line at the very bottom
	statusLine := m.renderStatusLine()

	// Combine main content with status line
	contentArea := lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusLine,
	)

	var result string
	if showSidebar {
		sidebar := m.renderSidebar(m.height - 2)
		result = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, contentArea)
	} else {
		result = contentArea
	}

	// Apply final height constraint to entire dashboard
	finalStyle := lipgloss.NewStyle().
		Height(m.height).
		MaxWidth(m.width)

	return finalStyle.Render(result)
}
