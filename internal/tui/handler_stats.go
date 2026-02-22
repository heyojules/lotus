package tui

import (
	"github.com/control-theory/lotus/internal/model"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// StatsModal displays comprehensive log statistics.
// It owns its own data fields (moved off DashboardModel).
type StatsModal struct {
	dashboard *DashboardModel
	viewport  viewport.Model

	// Data owned by this modal — only fetched while modal is visible.
	totalLogBytes  int64
	severityCounts map[string]int64
	hostStats      []model.DimensionCount
	serviceStats   []model.DimensionCount
	attributeStats []model.AttributeStat
}

func NewStatsModal(m *DashboardModel) *StatsModal {
	sm := &StatsModal{
		dashboard: m,
		viewport:  viewport.New(80, 20),
	}
	// Fetch data immediately on open.
	sm.Refresh()
	return sm
}

func (s *StatsModal) ID() string { return "stats" }

// Refresh implements Refreshable — called on each tick while this modal is visible.
func (s *StatsModal) Refresh() {
	store := s.dashboard.store
	if store == nil {
		return
	}
	opts := s.dashboard.queryOpts()
	if v, err := store.TotalLogBytes(opts); err == nil {
		s.totalLogBytes = v
	}
	if v, err := store.SeverityCounts(opts); err == nil {
		s.severityCounts = v
	}
	if v, err := store.TopHosts(20, opts); err == nil {
		s.hostStats = v
	}
	if v, err := store.TopServices(20, opts); err == nil {
		s.serviceStats = v
	}
	if v, err := store.TopAttributes(100, opts); err == nil {
		s.attributeStats = v
	}
}

func (s *StatsModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
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
		case "i", "escape", "esc":
			return true, nil
		}
		var cmd tea.Cmd
		s.viewport, cmd = s.viewport.Update(msg)
		return false, cmd

	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if s.dashboard.reverseScrollWheel {
					s.viewport.ScrollDown(1)
				} else {
					s.viewport.ScrollUp(1)
				}
				return false, nil
			case tea.MouseButtonWheelDown:
				if s.dashboard.reverseScrollWheel {
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

func (s *StatsModal) View(width, height int) string {
	return s.dashboard.renderStatsModalWithViewport(&s.viewport, s, width, height)
}
