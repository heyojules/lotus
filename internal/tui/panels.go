package tui

import (
	"github.com/tinytelemetry/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

// ChartPanel is a pluggable dashboard chart panel.
type ChartPanel interface {
	ID() string
	Title() string
	Refresh(store model.LogQuerier, opts model.QueryOpts) // fetch data on tick
	Render(ctx ViewContext, width, height int, active bool, selIdx int) string
	ContentLines(ctx ViewContext) int
	ItemCount() int
	OnSelect(ctx ViewContext, selIdx int) tea.Cmd // returns nil or ActionMsg
}
