package tui

import tea "github.com/charmbracelet/bubbletea"

var severityLevels = []string{"FATAL", "CRITICAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE", "UNKNOWN"}

// SeverityFilterModal displays a severity level filter selection.
type SeverityFilterModal struct {
	dashboard *DashboardModel
	selected  int
	original  map[string]bool // snapshot for ESC cancellation
}

func NewSeverityFilterModal(m *DashboardModel) *SeverityFilterModal {
	// Snapshot the current filter state.
	original := make(map[string]bool, len(m.severityFilter))
	for k, v := range m.severityFilter {
		original[k] = v
	}
	return &SeverityFilterModal{
		dashboard: m,
		selected:  0,
		original:  original,
	}
}

func (s *SeverityFilterModal) ID() string { return "severityfilter" }

func (s *SeverityFilterModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	m := s.dashboard
	switch msg := msg.(type) {
	case tea.KeyMsg:
		totalItems := len(severityLevels) + 3 // +3 for "Select All", "Select None", and separator

		switch msg.String() {
		case "ctrl+c":
			return false, tea.Quit
		case "up", "k":
			if s.selected > 0 {
				s.selected--
				if s.selected == 2 {
					s.selected = 1
				}
			}
			return false, nil
		case "down", "j":
			if s.selected < totalItems-1 {
				s.selected++
				if s.selected == 2 {
					s.selected = 3
				}
			}
			return false, nil
		case " ":
			if s.selected == 0 {
				for _, severity := range severityLevels {
					m.severityFilter[severity] = true
				}
			} else if s.selected == 1 {
				for _, severity := range severityLevels {
					m.severityFilter[severity] = false
				}
			} else if s.selected >= 3 {
				severityIndex := s.selected - 3
				if severityIndex < len(severityLevels) {
					severity := severityLevels[severityIndex]
					m.severityFilter[severity] = !m.severityFilter[severity]
				}
			}
			m.updateSeverityFilterActiveStatus()
			return false, nil
		case "enter":
			if s.selected == 0 {
				for _, severity := range severityLevels {
					m.severityFilter[severity] = true
				}
			} else if s.selected == 1 {
				for _, severity := range severityLevels {
					m.severityFilter[severity] = false
				}
			}
			m.updateSeverityFilterActiveStatus()
			return true, nil
		case "escape", "esc":
			// Restore original state (cancel changes)
			for k, v := range s.original {
				m.severityFilter[k] = v
			}
			m.updateSeverityFilterActiveStatus()
			return true, nil
		}
		return false, nil

	case tea.MouseMsg:
		return false, nil // swallow mouse events
	}
	return false, nil
}

func (s *SeverityFilterModal) View(width, height int) string {
	return s.dashboard.renderSeverityFilterModalView(s.selected, width, height)
}
