package tui

import (
	"fmt"
	"strings"

	"github.com/tinytelemetry/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PatternsChartPanel displays drain3 log patterns.
type PatternsChartPanel struct {
	drain3Manager *Drain3Manager
	pushModalCmd  tea.Cmd
}

// NewPatternsChartPanel creates a new patterns chart panel.
func NewPatternsChartPanel(m *DashboardModel) *PatternsChartPanel {
	return &PatternsChartPanel{
		drain3Manager: m.drain3Manager,
		pushModalCmd: func() tea.Msg {
			modal := NewPatternsModal(m)
			return ActionMsg{Action: ActionPushModal, Payload: modal}
		},
	}
}

func (p *PatternsChartPanel) ID() string    { return "patterns" }
func (p *PatternsChartPanel) Title() string { return "Patterns" }

func (p *PatternsChartPanel) Refresh(_ model.LogQuerier, _ model.QueryOpts) {
	// no-op: drain3Manager patterns are updated as logs arrive
}

func (p *PatternsChartPanel) ContentLines(_ ViewContext) int {
	return 8
}

func (p *PatternsChartPanel) ItemCount() int {
	return 7
}

func (p *PatternsChartPanel) Render(_ ViewContext, width, height int, active bool, _ int) string {
	style := sectionStyle.Width(width).Height(height)
	if active {
		style = activeSectionStyle.Width(width).Height(height)
	}

	patternCount, totalLogs := 0, 0
	if p.drain3Manager != nil {
		patternCount, totalLogs = p.drain3Manager.GetStats()
	}

	titleText := "Log Patterns"
	if patternCount > 0 {
		titleText = fmt.Sprintf("Log Patterns (%d patterns from %d logs)", patternCount, totalLogs)
	}
	title := chartTitleStyle.Render(titleText)

	var content string
	if p.drain3Manager != nil && patternCount > 0 {
		content = p.renderContent(width)
	} else {
		content = helpStyle.Render("Extracting patterns")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

func (p *PatternsChartPanel) OnSelect(_ ViewContext, _ int) tea.Cmd {
	if p.drain3Manager != nil && p.pushModalCmd != nil {
		return p.pushModalCmd
	}
	return nil
}

func (p *PatternsChartPanel) renderContent(chartWidth int) string {
	if p.drain3Manager == nil {
		return helpStyle.Render("Pattern extraction not available")
	}

	patterns := p.drain3Manager.GetTopPatterns(8)

	maxCount := 0
	for _, pat := range patterns {
		if pat.Count > maxCount {
			maxCount = pat.Count
		}
	}

	templateWidth := chartWidth - 26
	if templateWidth < 20 {
		templateWidth = 20
	}

	const displayLines = 8
	var lines []string

	for i := 0; i < displayLines; i++ {
		if i < len(patterns) {
			pattern := patterns[i]

			barWidth := 12
			fillWidth := int(float64(pattern.Count) * float64(barWidth) / float64(maxCount))
			if fillWidth == 0 && pattern.Count > 0 {
				fillWidth = 1
			}

			bar := strings.Repeat("█", fillWidth) + strings.Repeat("░", barWidth-fillWidth)
			percentage := fmt.Sprintf("%5.1f%%", pattern.Percentage)

			template := pattern.Template
			if len(template) > templateWidth {
				template = template[:templateWidth-3] + "..."
			}

			var barColor lipgloss.Style
			if i < 3 {
				barColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
			} else if i < 6 {
				barColor = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
			} else {
				barColor = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
			}

			line := fmt.Sprintf("%s %s │ %s",
				barColor.Render(bar),
				lipgloss.NewStyle().Foreground(ColorGray).Render(percentage),
				lipgloss.NewStyle().Foreground(ColorWhite).Render(template),
			)
			lines = append(lines, line)
		} else {
			emptyBar := strings.Repeat("░", 12)
			grayStyle := lipgloss.NewStyle().Foreground(ColorGray)
			line := fmt.Sprintf("%s %s  │ %s",
				grayStyle.Render(emptyBar),
				grayStyle.Render("     "),
				grayStyle.Render("(no pattern)"),
			)
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}
