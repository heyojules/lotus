package tui

import (
	"fmt"
	"strings"

	"github.com/tinytelemetry/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WordsChartPanel displays the most frequent words.
type WordsChartPanel struct {
	data []model.WordCount
}

// NewWordsChartPanel creates a new words chart panel.
func NewWordsChartPanel() *WordsChartPanel {
	return &WordsChartPanel{}
}

func (p *WordsChartPanel) ID() string    { return "words" }
func (p *WordsChartPanel) Title() string { return "Words" }

func (p *WordsChartPanel) Refresh(_ model.LogQuerier, _ model.QueryOpts) {
	// no-op: data is pushed from async tick results
}

func (p *WordsChartPanel) SetData(words []model.WordCount) {
	p.data = append([]model.WordCount(nil), words...)
}

func (p *WordsChartPanel) ContentLines(ctx ViewContext) int {
	minLines := 8
	if ctx.ContentWidth < 80 {
		minLines = 5
	}
	if len(p.data) == 0 {
		return minLines
	}
	maxItems := min(len(p.data), 10)
	if ctx.ContentWidth < 80 {
		maxItems = min(maxItems, 5)
	}
	return max(maxItems, minLines)
}

func (p *WordsChartPanel) ItemCount() int {
	return min(len(p.data), 10)
}

func (p *WordsChartPanel) Render(ctx ViewContext, width, height int, active bool, selIdx int) string {
	style := sectionStyle.Width(width).Height(height)
	if active {
		style = activeSectionStyle.Width(width).Height(height)
	}

	title := chartTitleStyle.Render("Top Words")

	var content string
	if len(p.data) > 0 {
		content = p.renderContent(ctx, width, selIdx, active)
	} else {
		content = helpStyle.Render("No data available")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

func (p *WordsChartPanel) OnSelect(ctx ViewContext, selIdx int) tea.Cmd {
	if selIdx < len(p.data) {
		entry := p.data[selIdx]
		newTerm := entry.Word
		if ctx.SearchTerm == entry.Word {
			newTerm = ""
		}
		return actionMsg(ActionMsg{Action: ActionSetSearchTerm, Payload: newTerm})
	}
	return nil
}

func (p *WordsChartPanel) renderContent(ctx ViewContext, chartWidth int, selectedIdx int, active bool) string {
	maxItems := 10
	if ctx.ContentWidth < 80 {
		maxItems = 5
	}
	if len(p.data) < maxItems {
		maxItems = len(p.data)
	}

	var lines []string

	maxCount := int64(0)
	for _, entry := range p.data {
		if entry.Count > maxCount {
			maxCount = entry.Count
		}
	}

	countFieldWidth := len(fmt.Sprintf("%d", maxCount))
	if countFieldWidth < 3 {
		countFieldWidth = 3
	}

	availableWidth := chartWidth - 2
	fixedOverhead := 4 + (countFieldWidth + 2) + 2
	barWidth := 15
	if availableWidth < 40 {
		barWidth = 8
	}

	labelWidth := availableWidth - fixedOverhead - barWidth
	if labelWidth < 8 {
		labelWidth = 8
	}

	for i := 0; i < maxItems; i++ {
		entry := p.data[i]

		topCount := p.data[0].Count
		filled := int((float64(entry.Count) / float64(topCount)) * float64(barWidth))
		if filled == 0 && entry.Count > 0 {
			filled = 1
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		formatStr := fmt.Sprintf("%%2d. %%-%ds %%%dd |%%s|", labelWidth, countFieldWidth)
		line := fmt.Sprintf(formatStr, i+1, entry.Word, entry.Count, bar)

		if i == selectedIdx && active {
			line = lipgloss.NewStyle().
				Background(ColorBlue).
				Foreground(ColorWhite).
				Render(line)
		} else {
			line = lipgloss.NewStyle().
				Foreground(ColorWhite).
				Render(line)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
