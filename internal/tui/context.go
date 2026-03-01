package tui

import tea "github.com/charmbracelet/bubbletea"

// ViewContext provides read-only context to decks for rendering,
// replacing direct access to *DashboardModel.
type ViewContext struct {
	ContentWidth   int
	ContentHeight  int
	SearchTerm     string
	SelectedApp    string
	UseLogTime     bool
	DeckPaused    bool   // per-deck pause state (set per render)
	DeckLastError string // per-deck last error (set per render)
	DeckLoading   bool   // true when deck's data fetch is in-flight
}

// ModalContext provides read-only context to modals for rendering, replacing
// direct access to *DashboardModel. Modals that need to render delegate to
// stored render callbacks, which capture the dashboard internally.
type ModalContext struct {
	ReverseScrollWheel bool
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
	Payload any
}

// actionMsg wraps ActionMsg as a tea.Msg.
func actionMsg(a ActionMsg) tea.Cmd {
	return func() tea.Msg { return a }
}
