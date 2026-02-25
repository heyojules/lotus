package tui

import (
	"github.com/tinytelemetry/lotus/internal/model"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// DetailModal displays detail content (log details or top values).
type DetailModal struct {
	dashboard    *DashboardModel
	viewport     viewport.Model
	content      string
	logEntry     *model.LogRecord // non-nil for log details view
}

func NewDetailModal(m *DashboardModel, entry *model.LogRecord) *DetailModal {
	dm := &DetailModal{
		dashboard: m,
		viewport:  viewport.New(80, 20),
		logEntry:  entry,
	}
	if entry != nil {
		dm.content = m.formatLogDetails(*entry, 60)
	}
	return dm
}

func NewDetailModalWithContent(m *DashboardModel, content string) *DetailModal {
	return &DetailModal{
		dashboard: m,
		viewport:  viewport.New(80, 20),
		content:   content,
	}
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
				if d.dashboard.reverseScrollWheel {
					d.viewport.ScrollDown(1)
				} else {
					d.viewport.ScrollUp(1)
				}
				return false, nil
			case tea.MouseButtonWheelDown:
				if d.dashboard.reverseScrollWheel {
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
	if d.logEntry != nil {
		return d.dashboard.renderSplitModalView(&d.viewport, d.logEntry, width, height)
	}
	return d.dashboard.renderSingleModalView(&d.viewport, d.content, width, height)
}
