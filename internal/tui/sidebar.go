package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
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

func (m *DashboardModel) moveSidebarCursor(delta int) tea.Cmd {
	items := m.sidebarItems()
	if len(items) == 0 {
		return nil
	}
	m.sidebarCursor += delta
	m.clampSidebarCursor()
	return m.applySidebarItem(items[m.sidebarCursor])
}

func (m *DashboardModel) activateSidebarCursor() tea.Cmd {
	items := m.sidebarItems()
	if len(items) == 0 {
		return nil
	}
	m.clampSidebarCursor()
	return m.applySidebarItem(items[m.sidebarCursor])
}

func (m *DashboardModel) applySidebarItem(item sidebarItem) tea.Cmd {
	switch item.kind {
	case sidebarItemPage:
		m.activatePage(item.pageIdx)
		return nil
	case sidebarItemApp:
		prev := m.selectedApp
		m.selectedApp = item.appName
		if prev == item.appName {
			return nil
		}
		// Immediately refresh logs and deck data for the new app.
		var cmds []tea.Cmd

		// Refresh logs.
		m.tickInFlight = false
		opts := m.queryOpts()
		severityLevels := m.activeSeverityLevels()
		var messagePattern string
		if m.filterRegex != nil {
			messagePattern = m.filterRegex.String()
		}
		logLimit := m.visibleLogLines()
		drainFrom := m.drain3LastProcessed
		cmds = append(cmds, m.fetchTickDataCmd(opts, severityLevels, messagePattern, logLimit, drainFrom))

		// Refresh decks.
		for tid, state := range m.deckStates {
			if state.FetchInFlight {
				continue
			}
			for _, vw := range m.allViews() {
				for _, dk := range vw.Decks {
					if tp, ok := dk.(TickableDeck); ok && tp.TypeID() == tid {
						state.FetchInFlight = true
						cmds = append(cmds, tp.FetchCmd(m.store, opts))
						goto nextDeck
					}
				}
			}
		nextDeck:
		}

		return tea.Batch(cmds...)
	}
	return nil
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
