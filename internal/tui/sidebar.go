package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

const sidebarWidth = 22

// renderSidebar renders the application sidebar.
func (m *DashboardModel) renderSidebar(height int) string {
	style := lipgloss.NewStyle().
		Width(sidebarWidth-2).
		Height(height).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorGray).
		Padding(0, 1)

	if m.activeSection == SectionSidebar {
		style = style.BorderForeground(ColorBlue)
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Apps"))
	lines = append(lines, "")

	// "All" entry at index 0
	allLabel := "  All"
	if m.selectedApp == "" {
		allLabel = "> All"
	}
	if m.activeSection == SectionSidebar && m.appListIdx == 0 {
		allLabel = lipgloss.NewStyle().Foreground(ColorBlue).Bold(true).Render(allLabel)
	}
	lines = append(lines, allLabel)

	if len(m.appList) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorGray).Render("  (no apps yet)"))
	}

	// Individual apps
	for i, app := range m.appList {
		idx := i + 1 // offset by 1 for "All"

		count := m.appCounts[app]

		label := fmt.Sprintf("  %s", app)
		if m.selectedApp == app {
			label = fmt.Sprintf("> %s", app)
		}

		countStr := fmt.Sprintf(" %d", count)
		maxLabelWidth := sidebarWidth - 6 - len(countStr)
		if len(label) > maxLabelWidth && maxLabelWidth > 3 {
			label = label[:maxLabelWidth-1] + "~"
		}
		label = label + countStr

		if m.activeSection == SectionSidebar && m.appListIdx == idx {
			label = lipgloss.NewStyle().Foreground(ColorBlue).Bold(true).Render(label)
		}

		lines = append(lines, label)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return style.Render(content)
}
