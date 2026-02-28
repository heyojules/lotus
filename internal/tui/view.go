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

// layoutHeights computes the three main vertical layout sections so that both
// renderDashboard and visibleLogLines share a single source of truth.
func (m *DashboardModel) layoutHeights() (decksHeight, filterHeight, logsHeight int) {
	statusLineHeight := 1
	usableHeight := m.height - statusLineHeight - 2

	filterHeight = 0
	if m.hasFilterOrSearch() {
		filterHeight = 1
	}

	// No decks → full-height logs (List view).
	if len(m.decks) == 0 {
		return 0, filterHeight, usableHeight - filterHeight
	}

	// Has decks → full-height decks (Base view).
	return usableHeight - filterHeight, filterHeight, 0
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
	// Ensure minimum dimensions
	if m.height < 20 || m.width < 60 {
		return "Terminal too small. Resize to at least 60x20."
	}

	// Determine content width (accounting for sidebar)
	contentWidth := m.contentWidth()
	showSidebar := m.sidebarVisible

	statusLineHeight := 1

	// Non-Logs pages with no decks: show placeholder.
	pg := m.activePage()
	isLogsPage := pg != nil && pg.ID == "logs"
	if len(m.decks) == 0 && !isLogsPage {
		placeholderHeight := m.height - statusLineHeight - 2
		placeholder := renderEmptyPagePlaceholder(m.currentPageTitle(), contentWidth, placeholderHeight)

		statusLine := m.renderStatusLine()
		contentArea := lipgloss.JoinVertical(lipgloss.Left, placeholder, statusLine)

		var result string
		if showSidebar {
			sidebar := m.renderSidebar(m.height - 2)
			result = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, contentArea)
		} else {
			result = contentArea
		}
		return m.viewStyle.Render(result)
	}

	decksHeight, _, logsHeight := m.layoutHeights()

	var sections []string

	// Decks grid (Base view — has decks).
	if len(m.decks) > 0 && decksHeight > 0 {
		topSection := m.renderDecksGrid(contentWidth, decksHeight)
		sections = append(sections, topSection)
	}

	// Filter bar (shown in both views when active).
	if m.hasFilterOrSearch() {
		filterSection := m.renderFilter()
		sections = append(sections, filterSection)
	}

	// Log scroll (List view — no decks).
	if logsHeight > 0 {
		logsSection := m.renderLogScroll(logsHeight)
		sections = append(sections, logsSection)
	}

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

	// Apply cached height/width constraint to entire dashboard
	return m.viewStyle.Render(result)
}

// renderEmptyPagePlaceholder renders a centered placeholder for pages with no decks.
func renderEmptyPagePlaceholder(title string, width, height int) string {
	heading := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("7")).
		Render(title)

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Coming soon")

	block := lipgloss.JoinVertical(lipgloss.Center, heading, subtitle)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, block)
}
