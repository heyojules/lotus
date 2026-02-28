package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// PatternsModal displays all log patterns.
type PatternsModal struct {
	ctx        ModalContext
	viewport   viewport.Model
	renderView func(vp *viewport.Model, width, height int) string
}

func NewPatternsModal(m *DashboardModel) *PatternsModal {
	return &PatternsModal{
		ctx:      m.modalContext(),
		viewport: viewport.New(80, 20),
		renderView: func(vp *viewport.Model, width, height int) string {
			return m.renderPatternsModalWithViewport(vp, width, height)
		},
	}
}

func (p *PatternsModal) ID() string { return "patterns" }

func (p *PatternsModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.viewport.ScrollUp(1)
			return false, nil
		case "down", "j":
			p.viewport.ScrollDown(1)
			return false, nil
		case "pgup":
			p.viewport.HalfPageUp()
			return false, nil
		case "pgdown":
			p.viewport.HalfPageDown()
			return false, nil
		case "escape", "esc":
			return true, nil
		}
		var cmd tea.Cmd
		p.viewport, cmd = p.viewport.Update(msg)
		return false, cmd

	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if p.ctx.ReverseScrollWheel {
					p.viewport.ScrollDown(1)
				} else {
					p.viewport.ScrollUp(1)
				}
				return false, nil
			case tea.MouseButtonWheelDown:
				if p.ctx.ReverseScrollWheel {
					p.viewport.ScrollUp(1)
				} else {
					p.viewport.ScrollDown(1)
				}
				return false, nil
			}
		}
		return false, nil
	}
	return false, nil
}

func (p *PatternsModal) View(width, height int) string {
	return p.renderView(&p.viewport, width, height)
}
