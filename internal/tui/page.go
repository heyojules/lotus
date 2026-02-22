package tui

import tea "github.com/charmbracelet/bubbletea"

// Page represents a top-level screen in the TUI (dashboard, settings, etc.).
type Page interface {
	ID() string
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Cmd, *PageNav)
	View(width, height int) string
}

// PageNav is returned from Update to request a page switch.
type PageNav struct {
	PageID string
	Params interface{}
}
