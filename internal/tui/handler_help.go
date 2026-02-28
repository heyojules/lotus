package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// HelpModal displays the help documentation.
type HelpModal struct {
	ctx        ModalContext
	viewport   viewport.Model
	renderView func(vp *viewport.Model, width, height int) string
}

func NewHelpModal(m *DashboardModel) *HelpModal {
	return &HelpModal{
		ctx:      m.modalContext(),
		viewport: viewport.New(80, 20),
		renderView: func(vp *viewport.Model, width, height int) string {
			return m.renderHelpModalWithViewport(vp, width, height)
		},
	}
}

func (h *HelpModal) ID() string { return "help" }

func (h *HelpModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			h.viewport.ScrollUp(1)
			return false, nil
		case "down", "j":
			h.viewport.ScrollDown(1)
			return false, nil
		case "pgup":
			h.viewport.HalfPageUp()
			return false, nil
		case "pgdown":
			h.viewport.HalfPageDown()
			return false, nil
		case "?", "h", "escape", "esc":
			return true, nil
		}
		var cmd tea.Cmd
		h.viewport, cmd = h.viewport.Update(msg)
		return false, cmd

	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if h.ctx.ReverseScrollWheel {
					h.viewport.ScrollDown(1)
				} else {
					h.viewport.ScrollUp(1)
				}
				return false, nil
			case tea.MouseButtonWheelDown:
				if h.ctx.ReverseScrollWheel {
					h.viewport.ScrollUp(1)
				} else {
					h.viewport.ScrollDown(1)
				}
				return false, nil
			}
		}
		return false, nil
	}
	return false, nil
}

func (h *HelpModal) View(width, height int) string {
	return h.renderView(&h.viewport, width, height)
}
