package tui

import (
	"github.com/tinytelemetry/tiny-telemetry/internal/model"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// SeverityModal displays a full-view severity timeline chart with time range selection.
type SeverityModal struct {
	ctx        ModalContext
	viewport   viewport.Model
	renderView func(vp *viewport.Model, sm *SeverityModal, width, height int) string
	refreshFn  func(sm *SeverityModal)

	// Time range selection: 0=1 day, 1=1 week, 2=1 month
	activeRange int
	rangeLabels []string

	// Data owned by this modal
	data []model.MinuteCounts
}

func NewSeverityModal(m *DashboardModel) *SeverityModal {
	sm := &SeverityModal{
		ctx:         m.modalContext(),
		viewport:    viewport.New(80, 20),
		activeRange: 0,
		rangeLabels: []string{"1 Day", "1 Week", "1 Month"},
		renderView: func(vp *viewport.Model, sm *SeverityModal, width, height int) string {
			return renderSeverityModalView(vp, sm, width, height)
		},
		refreshFn: func(sm *SeverityModal) {
			store := m.store
			if store == nil {
				return
			}
			opts := m.queryOpts()
			if rows, err := store.SeverityCountsByMinute(opts); err == nil {
				sm.data = rows
			}
		},
	}
	sm.Refresh()
	return sm
}

func (s *SeverityModal) ID() string { return "severity-timeline" }

func (s *SeverityModal) Refresh() {
	s.refreshFn(s)
}

func (s *SeverityModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "escape", "esc":
			return true, nil
		case "tab", "right", "l":
			s.activeRange = (s.activeRange + 1) % len(s.rangeLabels)
			return false, nil
		case "shift+tab", "left", "h":
			s.activeRange = (s.activeRange - 1 + len(s.rangeLabels)) % len(s.rangeLabels)
			return false, nil
		case "1":
			s.activeRange = 0
			return false, nil
		case "2":
			s.activeRange = 1
			return false, nil
		case "3":
			s.activeRange = 2
			return false, nil
		case "up", "k":
			s.viewport.ScrollUp(1)
			return false, nil
		case "down", "j":
			s.viewport.ScrollDown(1)
			return false, nil
		case "pgup":
			s.viewport.HalfPageUp()
			return false, nil
		case "pgdown":
			s.viewport.HalfPageDown()
			return false, nil
		}
		var cmd tea.Cmd
		s.viewport, cmd = s.viewport.Update(msg)
		return false, cmd

	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if s.ctx.ReverseScrollWheel {
					s.viewport.ScrollDown(1)
				} else {
					s.viewport.ScrollUp(1)
				}
				return false, nil
			case tea.MouseButtonWheelDown:
				if s.ctx.ReverseScrollWheel {
					s.viewport.ScrollUp(1)
				} else {
					s.viewport.ScrollDown(1)
				}
				return false, nil
			}
		}
		return false, nil
	}
	return false, nil
}

func (s *SeverityModal) View(width, height int) string {
	return s.renderView(&s.viewport, s, width, height)
}
