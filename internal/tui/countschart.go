package tui

import (
	"fmt"
	"strings"

	"github.com/control-theory/lotus/internal/model"

	"github.com/NimbleMarkets/ntcharts/barchart"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CountsChartPanel displays log counts over time as a stacked bar chart.
type CountsChartPanel struct {
	pushModalCmd tea.Cmd
	data         []SeverityCounts
}

// NewCountsChartPanel creates a new counts chart panel.
func NewCountsChartPanel(m *DashboardModel) *CountsChartPanel {
	return &CountsChartPanel{
		pushModalCmd: func() tea.Msg {
			modal := NewCountsModal(m)
			return ActionMsg{Action: ActionPushModal, Payload: modal}
		},
		data: make([]SeverityCounts, 0),
	}
}

func (p *CountsChartPanel) ID() string    { return "counts" }
func (p *CountsChartPanel) Title() string { return "Counts" }

func (p *CountsChartPanel) Refresh(_ model.LogQuerier, _ model.QueryOpts) {
	// no-op: data is pushed from async tick results
}

func (p *CountsChartPanel) SetData(history []SeverityCounts) {
	p.data = append([]SeverityCounts(nil), history...)
}

func (p *CountsChartPanel) ContentLines(ctx ViewContext) int {
	if len(p.data) == 0 {
		return 1
	}
	chartHeight := 8
	if ctx.ContentWidth < 80 {
		chartHeight = 6
	}
	return chartHeight
}

func (p *CountsChartPanel) ItemCount() int {
	return len(p.data)
}

func (p *CountsChartPanel) Render(ctx ViewContext, width, height int, active bool, _ int) string {
	style := sectionStyle.Width(width).Height(height)
	if active {
		style = activeSectionStyle.Width(width).Height(height)
	}

	var headerText string
	if len(p.data) > 0 {
		latest := p.data[len(p.data)-1]
		minTotal, maxTotal := latest.Total, latest.Total
		for _, counts := range p.data {
			if counts.Total < minTotal {
				minTotal = counts.Total
			}
			if counts.Total > maxTotal {
				maxTotal = counts.Total
			}
		}
		leftTitle := "Log Counts"
		rightStats := fmt.Sprintf("Min: %d | Max: %d", minTotal, maxTotal)
		availableWidth := width - 4
		spacerWidth := availableWidth - len(leftTitle) - len(rightStats)
		if spacerWidth > 0 {
			headerText = leftTitle + strings.Repeat(" ", spacerWidth) + rightStats
		} else {
			headerText = leftTitle
		}
	} else {
		headerText = "Log Counts"
	}

	title := chartTitleStyle.Render(headerText)

	var content string
	if len(p.data) > 0 {
		content = p.renderContent(ctx, width)
	} else {
		content = helpStyle.Render("No data available")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

func (p *CountsChartPanel) OnSelect(_ ViewContext, _ int) tea.Cmd {
	if p.pushModalCmd == nil {
		return nil
	}
	return p.pushModalCmd
}

func (p *CountsChartPanel) renderContent(ctx ViewContext, chartWidth int) string {
	if len(p.data) == 0 {
		return helpStyle.Render("No data available")
	}

	totalLogs := 0
	for _, counts := range p.data {
		totalLogs += counts.Total
	}

	legendWidth := 18
	chartHeight := 8
	actualChartWidth := chartWidth - legendWidth - 2
	if actualChartWidth < 20 {
		actualChartWidth = 20
	}
	if ctx.ContentWidth < 80 {
		chartHeight = 6
	}

	dataPoints := len(p.data)
	maxBars := actualChartWidth / 3

	var paddingCount int
	var dataStartIdx int

	if dataPoints < maxBars {
		paddingCount = maxBars - dataPoints
		dataStartIdx = 0
	} else {
		paddingCount = 0
		dataStartIdx = dataPoints - maxBars
	}

	bc := barchart.New(actualChartWidth, chartHeight,
		barchart.WithBarGap(1),
		barchart.WithBarWidth(1),
		barchart.WithNoAxis(),
	)

	severityColors := map[string]lipgloss.Style{
		"TRACE":    lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Background(lipgloss.Color("240")),
		"DEBUG":    lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Background(lipgloss.Color("244")),
		"INFO":     lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Background(lipgloss.Color("39")),
		"WARN":     lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Background(lipgloss.Color("208")),
		"ERROR":    lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Background(lipgloss.Color("196")),
		"FATAL":    lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Background(lipgloss.Color("201")),
		"CRITICAL": lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Background(lipgloss.Color("201")),
		"UNKNOWN":  lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("250")),
	}

	for i := 0; i < paddingCount; i++ {
		bc.Push(barchart.BarData{
			Label: "",
			Values: []barchart.BarValue{
				{Name: "EMPTY", Value: 0, Style: severityColors["UNKNOWN"]},
			},
		})
	}

	actualDataCount := min(dataPoints, maxBars-paddingCount)
	for i := 0; i < actualDataCount; i++ {
		counts := p.data[dataStartIdx+i]

		var barValues []barchart.BarValue

		severityData := []struct {
			name  string
			count int
			style lipgloss.Style
		}{
			{"TRACE", counts.Trace, severityColors["TRACE"]},
			{"DEBUG", counts.Debug, severityColors["DEBUG"]},
			{"INFO", counts.Info, severityColors["INFO"]},
			{"WARN", counts.Warn, severityColors["WARN"]},
			{"ERROR", counts.Error, severityColors["ERROR"]},
			{"FATAL", counts.Fatal + counts.Critical, severityColors["FATAL"]},
		}

		for _, sev := range severityData {
			if sev.count > 0 {
				barValues = append(barValues, barchart.BarValue{
					Name:  sev.name,
					Value: float64(sev.count),
					Style: sev.style,
				})
			}
		}

		if len(barValues) == 0 {
			barValues = append(barValues, barchart.BarValue{
				Name:  "EMPTY",
				Value: 0.0,
				Style: severityColors["UNKNOWN"],
			})
		}

		bc.Push(barchart.BarData{Label: "", Values: barValues})
	}

	bc.Draw()
	chartOutput := bc.View()

	var legend string
	if len(p.data) > 0 {
		latest := p.data[len(p.data)-1]

		severityLevels := []struct {
			name  string
			count int
			color string
		}{
			{"FATAL", latest.Fatal + latest.Critical, "201"},
			{"ERROR", latest.Error, "196"},
			{"WARN", latest.Warn, "208"},
			{"INFO", latest.Info, "39"},
			{"DEBUG", latest.Debug, "244"},
			{"TRACE", latest.Trace, "240"},
			{"─────", 0, "7"},
			{"TOTAL", latest.Total, "7"},
		}

		var legendLines []string
		for _, sev := range severityLevels {
			if sev.name == "─────" {
				colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(sev.color))
				legendLines = append(legendLines, colorStyle.Render("─────────────"))
			} else {
				label := fmt.Sprintf("%-6s:", sev.name)
				value := fmt.Sprintf("%6d", sev.count)
				colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(sev.color))
				legendLines = append(legendLines, colorStyle.Render(label+value))
			}
		}

		for len(legendLines) < chartHeight {
			legendLines = append(legendLines, strings.Repeat(" ", legendWidth-2))
		}

		legend = strings.Join(legendLines, "\n")
	} else {
		legend = strings.Repeat("\n", chartHeight-1)
	}

	separator := strings.Repeat(" ", 2)
	chartLines := strings.Split(chartOutput, "\n")
	for len(chartLines) < chartHeight {
		chartLines = append(chartLines, "")
	}

	var combinedLines []string
	legendSplit := strings.Split(legend, "\n")

	for i := 0; i < chartHeight; i++ {
		chartLine := ""
		legendLine := ""
		if i < len(chartLines) {
			chartLine = chartLines[i]
		}
		if i < len(legendSplit) {
			legendLine = legendSplit[i]
		}
		if len(chartLine) < actualChartWidth {
			chartLine += strings.Repeat(" ", actualChartWidth-len(chartLine))
		}
		combinedLines = append(combinedLines, chartLine+separator+legendLine)
	}

	return strings.Join(combinedLines, "\n")
}
