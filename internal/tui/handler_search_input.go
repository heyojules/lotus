package tui

import tea "github.com/charmbracelet/bubbletea"

type searchInputHandler struct{}

func (h searchInputHandler) HandleKey(m *DashboardModel, msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return true, tea.Quit
	case "escape", "esc":
		m.searchActive = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.searchTerm = ""
		if m.activeSection == SectionFilter {
			m.activeSection = SectionCharts
			if m.activePanelIdx >= len(m.panels) {
				m.activePanelIdx = max(0, len(m.panels)-1)
			}
		}
		return true, nil
	case "enter":
		m.searchActive = false
		m.searchInput.Blur()
		m.searchTerm = m.searchInput.Value()
		m.activeSection = SectionLogs
		return true, nil
	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.searchTerm = m.searchInput.Value()
		return true, cmd
	}
}

func (h searchInputHandler) HandleMouse(_ *DashboardModel, _ tea.MouseMsg) (bool, tea.Cmd) {
	return true, nil // swallow mouse events during search input
}
