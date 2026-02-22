package tui

import (
	"time"

	"github.com/control-theory/lotus/internal/model"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// CountsModal displays log counts analysis with heatmap and services.
// It owns its own data fields (moved off DashboardModel).
type CountsModal struct {
	dashboard *DashboardModel
	viewport  viewport.Model

	// Data owned by this modal — only fetched while modal is visible.
	countsHeatmapData  []model.MinuteCounts
	countsServicesData map[string][]model.DimensionCount
}

func NewCountsModal(m *DashboardModel) *CountsModal {
	cm := &CountsModal{
		dashboard: m,
		viewport:  viewport.New(80, 20),
	}
	// Fetch data immediately on open.
	cm.Refresh()
	return cm
}

func (c *CountsModal) ID() string { return "counts" }

// Refresh implements Refreshable — called on each tick while this modal is visible.
func (c *CountsModal) Refresh() {
	store := c.dashboard.store
	if store == nil {
		return
	}
	opts := c.dashboard.queryOpts()

	// Heatmap data
	if rows, err := store.SeverityCountsByMinute(60*time.Minute, opts); err == nil {
		c.countsHeatmapData = rows
	}

	// Services by severity
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	servicesData := make(map[string][]model.DimensionCount, len(severities))
	for _, severity := range severities {
		if services, err := store.TopServicesBySeverity(severity, 3, opts); err == nil {
			servicesData[severity] = services
		}
	}
	c.countsServicesData = servicesData
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
				if c.dashboard.reverseScrollWheel {
					c.viewport.ScrollDown(1)
				} else {
					c.viewport.ScrollUp(1)
				}
				return false, nil
			case tea.MouseButtonWheelDown:
				if c.dashboard.reverseScrollWheel {
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
	return c.dashboard.renderCountsModalWithViewport(&c.viewport, c, width, height)
}
