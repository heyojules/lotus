package tui

import tea "github.com/charmbracelet/bubbletea"

// ViewContext provides read-only context to chart panels for rendering,
// replacing direct access to *DashboardModel.
type ViewContext struct {
	ContentWidth  int
	ContentHeight int
	SearchTerm    string
	SelectedApp   string
	UseLogTime    bool
}

// Action identifies what a panel wants the dashboard to do.
type Action int

const (
	ActionSetSearchTerm Action = iota
	ActionPushModal
)

// ActionMsg is returned by panel OnSelect to communicate with the dashboard
// without mutating it directly.
type ActionMsg struct {
	Action  Action
	Payload interface{}
}

// actionMsg wraps ActionMsg as a tea.Msg.
func actionMsg(a ActionMsg) tea.Cmd {
	return func() tea.Msg { return a }
}
