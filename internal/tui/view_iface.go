package tui

import tea "github.com/charmbracelet/bubbletea"

// View represents a top-level screen in the TUI (dashboard, settings, etc.).
type View interface {
	ID() string
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Cmd, *ViewNav)
	View(width, height int) string
}

// ViewNav is returned from Update to request a view switch.
type ViewNav struct {
	ViewID string
	Params interface{}
}
