package tui

import (
	"github.com/tinytelemetry/lotus/internal/model"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// CountsModal displays log counts analysis with heatmap and services.
// It owns its own data fields (moved off DashboardModel).
type CountsModal struct {
	ctx        ModalContext
	viewport   viewport.Model
	renderView func(vp *viewport.Model, cm *CountsModal, width, height int) string
	refreshFn  func(cm *CountsModal)

	// Data owned by this modal — only fetched while modal is visible.
	countsHeatmapData  []model.MinuteCounts
	countsServicesData map[string][]model.DimensionCount
}

func NewCountsModal(m *DashboardModel) *CountsModal {
	cm := &CountsModal{
		ctx:      m.modalContext(),
		viewport: viewport.New(80, 20),
		renderView: func(vp *viewport.Model, cm *CountsModal, width, height int) string {
			return m.renderCountsModalWithViewport(vp, cm, width, height)
		},
		refreshFn: func(cm *CountsModal) {
			store := m.store
			if store == nil {
				return
			}
			opts := m.queryOpts()

			if rows, err := store.SeverityCountsByMinute(opts); err == nil {
				cm.countsHeatmapData = rows
			}

			severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
			servicesData := make(map[string][]model.DimensionCount, len(severities))
			for _, severity := range severities {
				if services, err := store.TopServicesBySeverity(severity, 3, opts); err == nil {
					servicesData[severity] = services
				}
			}
			cm.countsServicesData = servicesData
		},
	}
	// Fetch data immediately on open.
	cm.Refresh()
	return cm
}

func (c *CountsModal) ID() string { return "counts" }

// Refresh implements Refreshable — called on each tick while this modal is visible.
func (c *CountsModal) Refresh() {
	c.refreshFn(c)
}

func (c *CountsModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			c.viewport.ScrollUp(1)
			return false, nil
		case "down", "j":
			c.viewport.ScrollDown(1)
			return false, nil
		case "pgup":
			c.viewport.HalfPageUp()
			return false, nil
		case "pgdown":
			c.viewport.HalfPageDown()
			return false, nil
		case "escape", "esc":
			return true, nil
		}
		var cmd tea.Cmd
		c.viewport, cmd = c.viewport.Update(msg)
		return false, cmd

	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if c.ctx.ReverseScrollWheel {
					c.viewport.ScrollDown(1)
				} else {
					c.viewport.ScrollUp(1)
				}
				return false, nil
			case tea.MouseButtonWheelDown:
				if c.ctx.ReverseScrollWheel {
					c.viewport.ScrollUp(1)
				} else {
					c.viewport.ScrollDown(1)
				}
				return false, nil
			}
		}
		return false, nil
	}
	return false, nil
}

func (c *CountsModal) View(width, height int) string {
	return c.renderView(&c.viewport, c, width, height)
}
