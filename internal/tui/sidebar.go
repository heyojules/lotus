package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

const sidebarWidth = 22

type sidebarItemKind int

const (
	sidebarItemPage sidebarItemKind = iota
	sidebarItemApp
)

type sidebarItem struct {
	kind    sidebarItemKind
	pageIdx int
	appName string // empty means "All"
}

func (m *DashboardModel) sidebarItems() []sidebarItem {
	items := make([]sidebarItem, 0, len(m.pages)+len(m.appList))

	for _, app := range m.appList {
		items = append(items, sidebarItem{
			kind:    sidebarItemApp,
			appName: app,
		})
	}

	for i := range m.pages {
		items = append(items, sidebarItem{
			kind:    sidebarItemPage,
			pageIdx: i,
		})
	}

	return items
}

func (m *DashboardModel) clampSidebarCursor() {
	items := m.sidebarItems()
	if len(items) == 0 {
		m.sidebarCursor = 0
		return
	}
	if m.sidebarCursor < 0 {
		m.sidebarCursor = 0
	}
	if m.sidebarCursor >= len(items) {
		m.sidebarCursor = len(items) - 1
	}
}

func (m *DashboardModel) moveSidebarCursor(delta int) {
	items := m.sidebarItems()
	if len(items) == 0 {
		return
	}
	m.sidebarCursor += delta
	m.clampSidebarCursor()
}

func (m *DashboardModel) activateSidebarCursor() {
	items := m.sidebarItems()
	if len(items) == 0 {
		return
	}
	m.clampSidebarCursor()
	m.applySidebarItem(items[m.sidebarCursor])
}

func (m *DashboardModel) applySidebarItem(item sidebarItem) {
	switch item.kind {
	case sidebarItemPage:
		m.activatePage(item.pageIdx)
	case sidebarItemApp:
		m.selectedApp = item.appName
	}
}

func (m *DashboardModel) buildSidebarLines() ([]string, map[int]int) {
	rowToCursor := make(map[int]int)
	lines := make([]string, 0, len(m.pages)+len(m.appList)+8)

	appendLine := func(line string) {
		lines = append(lines, line)
	}

	cursor := 0

	appendLine(lipgloss.NewStyle().Bold(true).Render("Apps"))
	appendLine("")

	if len(m.appList) == 0 {
		appendLine(lipgloss.NewStyle().Foreground(ColorGray).Render("  (no apps yet)"))
	}

	for _, app := range m.appList {
		label := fmt.Sprintf("  %s", app)
		if m.selectedApp == app {
			label = fmt.Sprintf("> %s", app)
		}

		maxLabelWidth := sidebarWidth - 4
		if len(label) > maxLabelWidth && maxLabelWidth > 3 {
			label = label[:maxLabelWidth-1] + "~"
		}

		rowToCursor[len(lines)] = cursor
		if m.activeSection == SectionSidebar && m.sidebarCursor == cursor {
			label = lipgloss.NewStyle().Foreground(ColorBlue).Bold(true).Render(label)
		}
		appendLine(label)
		cursor++
	}

	appendLine("")
	appendLine(lipgloss.NewStyle().Bold(true).Render("Pages"))
	appendLine("")

	for i, pg := range m.pages {
		label := fmt.Sprintf("  %s", pg.Title)
		if m.activePageIdx == i {
			label = fmt.Sprintf("> %s", pg.Title)
		}
		rowToCursor[len(lines)] = cursor
		if m.activeSection == SectionSidebar && m.sidebarCursor == cursor {
			label = lipgloss.NewStyle().Foreground(ColorBlue).Bold(true).Render(label)
		}
		appendLine(label)
		cursor++
	}

	return lines, rowToCursor
}

func (m *DashboardModel) sidebarCursorAtMouseRow(y int) (int, bool) {
	_, rowToCursor := m.buildSidebarLines()

	// Bubble Tea mouse row can include border/padding rows depending on renderer.
	for _, offset := range []int{0, -1, -2, 1} {
		row := y + offset
		if row < 0 {
			continue
		}
		if idx, ok := rowToCursor[row]; ok {
			return idx, true
		}
	}
	return 0, false
}

// renderSidebar renders page/app navigation in the left sidebar.
func (m *DashboardModel) renderSidebar(height int) string {
	m.clampSidebarCursor()

	style := lipgloss.NewStyle().
		Width(sidebarWidth-2).
		Height(height).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorGray).
		Padding(0, 1)

	if m.activeSection == SectionSidebar {
		style = style.BorderForeground(ColorBlue)
	}

	lines, _ := m.buildSidebarLines()
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return style.Render(content)
}
