package tui

import (
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
)

type filterInputHandler struct{}

func (h filterInputHandler) HandleKey(m *DashboardModel, msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return true, tea.Quit
	case "escape", "esc":
		m.filterActive = false
		m.filterInput.Blur()
		m.filterInput.SetValue("")
		m.filterRegex = nil
		if m.activeSection == SectionFilter {
			m.activeSection = SectionDecks
			if m.activeDeckIdx >= len(m.decks) {
				m.activeDeckIdx = max(0, len(m.decks)-1)
			}
		}
		return true, nil
	case "enter":
		m.filterActive = false
		m.filterInput.Blur()
		m.activeSection = SectionLogs
		return true, nil
	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		if m.filterInput.Value() != "" {
			if regex, err := regexp.Compile(m.filterInput.Value()); err == nil {
				m.filterRegex = regex
			}
		} else {
			m.filterRegex = nil
		}
		return true, cmd
	}
}

func (h filterInputHandler) HandleMouse(_ *DashboardModel, _ tea.MouseMsg) (bool, tea.Cmd) {
	return true, nil // swallow mouse events during filter input
}
