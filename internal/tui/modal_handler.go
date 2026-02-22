package tui

import tea "github.com/charmbracelet/bubbletea"

// Modal is a self-contained modal that owns its own Update/View lifecycle.
// Modals are managed via a stack on DashboardModel — the topmost modal
// receives all input and renders full-screen.
type Modal interface {
	// ID returns a unique identifier used to deduplicate pushes.
	ID() string
	// Update processes a message. Return pop=true to close the modal.
	Update(msg tea.Msg) (pop bool, cmd tea.Cmd)
	// View renders the modal content for the given terminal dimensions.
	View(width, height int) string
}

// Refreshable is optionally implemented by modals that need periodic data
// refresh while they are visible (i.e. on top of the stack).
type Refreshable interface {
	Refresh()
}

// ModalHandler handles key and mouse events for a specific modal or input mode.
// Used only by filter/search input handlers which are NOT modals (they are
// inline dashboard state).
type ModalHandler interface {
	// HandleKey processes a key press. Return handled=true if consumed.
	HandleKey(m *DashboardModel, msg tea.KeyMsg) (handled bool, cmd tea.Cmd)
	// HandleMouse processes mouse events. Return handled=true if consumed.
	HandleMouse(m *DashboardModel, msg tea.MouseMsg) (handled bool, cmd tea.Cmd)
}

// inlineHandlerEntry pairs an activation predicate with an inline handler.
// Only used for filter/search input which are part of the dashboard layout.
type inlineHandlerEntry struct {
	isActive func(m *DashboardModel) bool
	handler  ModalHandler
}
