package tui

import (
	"github.com/tinytelemetry/lotus/internal/model"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// DetailModal displays detail content (log details or top values).
type DetailModal struct {
	ctx        ModalContext
	viewport   viewport.Model
	content    string
	logEntry   *model.LogRecord // non-nil for log details view
	renderView func(vp *viewport.Model, width, height int) string
}

func NewDetailModal(m *DashboardModel, entry *model.LogRecord) *DetailModal {
	dm := &DetailModal{
		ctx:      m.modalContext(),
		viewport: viewport.New(80, 20),
		logEntry: entry,
	}
	if entry != nil {
		dm.content = m.formatLogDetails(*entry, 60)
		dm.renderView = func(vp *viewport.Model, width, height int) string {
			return m.renderSplitModalView(vp, dm.logEntry, width, height)
		}
	} else {
		dm.renderView = func(vp *viewport.Model, width, height int) string {
			return m.renderSingleModalView(vp, dm.content, width, height)
		}
	}
	return dm
}

func NewDetailModalWithContent(m *DashboardModel, content string) *DetailModal {
	dm := &DetailModal{
		ctx:      m.modalContext(),
		viewport: viewport.New(80, 20),
		content:  content,
	}
	dm.renderView = func(vp *viewport.Model, width, height int) string {
		return m.renderSingleModalView(vp, dm.content, width, height)
	}
	return dm
}

func (d *DetailModal) ID() string { return "detail" }

func (d *DetailModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			d.viewport.ScrollUp(1)
			return false, nil
		case "down", "j":
			d.viewport.ScrollDown(1)
			return false, nil
		case "pgup":
			d.viewport.HalfPageUp()
			return false, nil
		case "pgdown":
			d.viewport.HalfPageDown()
			return false, nil
		case "escape", "esc":
			return true, nil
		}
		var cmd tea.Cmd
		d.viewport, cmd = d.viewport.Update(msg)
		return false, cmd

	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if d.ctx.ReverseScrollWheel {
					d.viewport.ScrollDown(1)
				} else {
					d.viewport.ScrollUp(1)
				}
				return false, nil
			case tea.MouseButtonWheelDown:
				if d.ctx.ReverseScrollWheel {
					d.viewport.ScrollUp(1)
				} else {
					d.viewport.ScrollDown(1)
				}
				return false, nil
			}
		}
		return false, nil
	}
	return false, nil
}

func (d *DetailModal) View(width, height int) string {
	return d.renderView(&d.viewport, width, height)
}
