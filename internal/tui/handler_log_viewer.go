package tui

import tea "github.com/charmbracelet/bubbletea"

// LogViewerModal displays a fullscreen log viewer.
// It references shared logEntries via the dashboard pointer.
type LogViewerModal struct {
	dashboard *DashboardModel
}

func NewLogViewerModal(m *DashboardModel) *LogViewerModal {
	return &LogViewerModal{dashboard: m}
}

func (l *LogViewerModal) ID() string { return "logviewer" }

func (l *LogViewerModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	m := l.dashboard
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedLogIndex > 0 {
				m.selectedLogIndex--
			}
			return false, nil
		case "down", "j":
			if m.selectedLogIndex < len(m.logEntries)-1 {
				m.selectedLogIndex++
			}
			return false, nil
		case "pgup":
			m.selectedLogIndex = max(0, m.selectedLogIndex-10)
			return false, nil
		case "pgdown":
			m.selectedLogIndex = min(len(m.logEntries)-1, m.selectedLogIndex+10)
			return false, nil
		case "home":
			m.selectedLogIndex = 0
			return false, nil
		case "end":
			if len(m.logEntries) > 0 {
				m.selectedLogIndex = len(m.logEntries) - 1
			}
			return false, nil
		case "enter":
			if m.selectedLogIndex >= 0 && m.selectedLogIndex < len(m.logEntries) {
				entry := m.logEntries[m.selectedLogIndex]
				m.PushModal(NewDetailModal(m, &entry))
			}
			return false, nil
		case "/":
			m.PopModal() // close log viewer
			m.activeSection = SectionFilter
			m.filterActive = true
			m.filterInput.Focus()
			return true, nil
		case "s":
			m.PopModal() // close log viewer
			m.activeSection = SectionFilter
			m.searchActive = true
			m.searchInput.Focus()
			return true, nil
		case "c":
			m.showColumns = !m.showColumns
			return false, nil
		case "escape", "esc", "f":
			return true, nil
		}
		return false, nil // consume all keys while log viewer is open

	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if m.reverseScrollWheel {
					if m.selectedLogIndex > 0 {
						m.selectedLogIndex--
					}
				} else {
					if m.selectedLogIndex < len(m.logEntries)-1 {
						m.selectedLogIndex++
					}
				}
				return false, nil
			case tea.MouseButtonWheelDown:
				if m.reverseScrollWheel {
					if m.selectedLogIndex < len(m.logEntries)-1 {
						m.selectedLogIndex++
					}
				} else {
					if m.selectedLogIndex > 0 {
						m.selectedLogIndex--
					}
				}
				return false, nil
			}
		}
		return false, nil
	}
	return false, nil
}

func (l *LogViewerModal) View(width, height int) string {
	return l.dashboard.renderLogViewerModalView(width, height)
}
