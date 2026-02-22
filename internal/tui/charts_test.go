package tui

import (
	"testing"
	"time"

	"github.com/control-theory/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

// testPanel is a minimal ChartPanel for testing.
type testPanel struct {
	id    string
	title string
}

func (p *testPanel) ID() string    { return p.id }
func (p *testPanel) Title() string { return p.title }
func (p *testPanel) Refresh(_ model.LogQuerier, _ model.QueryOpts)              {}
func (p *testPanel) Render(_ ViewContext, _, _ int, _ bool, _ int) string          { return "test" }
func (p *testPanel) ContentLines(_ ViewContext) int                                { return 6 }
func (p *testPanel) ItemCount() int                                                { return 1 }
func (p *testPanel) OnSelect(_ ViewContext, _ int) tea.Cmd                         { return nil }

func TestChartLayout_AllowsMoreThanFourPanels(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")

	extra := &testPanel{id: "extra", title: "Extra"}

	panels := append([]ChartPanel{}, m.panels...)
	panels = append(panels, extra)
	m.SetPanels(panels)

	if got := len(m.panels); got != 5 {
		t.Fatalf("panel count = %d, want 5", got)
	}

	h := m.calculateRequiredChartsHeight()
	if h <= 0 {
		t.Fatalf("calculateRequiredChartsHeight = %d, want > 0", h)
	}

	view := m.renderChartsGrid(120, h)
	if view == "No panels registered" {
		t.Fatal("expected rendered grid for 5 panels")
	}
}

func TestChartPanelAt_ResolvesPanelIndex(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")
	m.width = 120
	m.height = 40

	idx, ok := m.chartPanelAt(120, m.calculateRequiredChartsHeight(), 0, 0)
	if !ok {
		t.Fatal("chartPanelAt should resolve top-left panel")
	}
	if idx != 0 {
		t.Fatalf("chartPanelAt top-left index = %d, want 0", idx)
	}
}
