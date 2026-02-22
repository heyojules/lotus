package tui

import (
	"fmt"
	"strings"

	"github.com/control-theory/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AttributesChartPanel displays the most frequent attribute keys.
type AttributesChartPanel struct {
	dashboard *DashboardModel // kept for formatAttributeValuesModal; TODO: extract
	data      []AttributeEntry
}

// NewAttributesChartPanel creates a new attributes chart panel.
func NewAttributesChartPanel(m *DashboardModel) *AttributesChartPanel {
	return &AttributesChartPanel{dashboard: m}
}

func (p *AttributesChartPanel) ID() string    { return "attributes" }
func (p *AttributesChartPanel) Title() string { return "Attrs" }

func (p *AttributesChartPanel) Refresh(store model.LogQuerier, opts model.QueryOpts) {
	if store == nil {
		return
	}
	attrKeys, err := store.TopAttributeKeys(20, opts)
	if err != nil {
		return
	}
	entries := make([]AttributeEntry, len(attrKeys))
	for i, ak := range attrKeys {
		values, _ := store.AttributeKeyValues(ak.Key, 100)
		entries[i] = AttributeEntry{
			Key:              ak.Key,
			UniqueValueCount: ak.UniqueValues,
			TotalCount:       ak.TotalCount,
			Values:           values,
		}
	}
	p.data = entries
}

func (p *AttributesChartPanel) ContentLines(ctx ViewContext) int {
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

func (p *AttributesChartPanel) ItemCount() int {
	return min(len(p.data), 10)
}

func (p *AttributesChartPanel) Render(ctx ViewContext, width, height int, active bool, selIdx int) string {
	style := sectionStyle.Width(width).Height(height)
	if active {
		style = activeSectionStyle.Width(width).Height(height)
	}

	title := chartTitleStyle.Render("Top Attributes")

	var content string
	if len(p.data) > 0 {
		content = p.renderContent(ctx, width, selIdx, active)
	} else {
		content = helpStyle.Render("No data available")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

func (p *AttributesChartPanel) OnSelect(ctx ViewContext, selIdx int) tea.Cmd {
	if selIdx < len(p.data) {
		entry := &p.data[selIdx]
		contentWidth := ctx.ContentWidth - 16
		if contentWidth < 60 {
			contentWidth = 60
		}
		content := p.dashboard.formatAttributeValuesModal(entry, contentWidth)
		modal := NewDetailModalWithContent(p.dashboard, content)
		return actionMsg(ActionMsg{Action: ActionPushModal, Payload: modal})
	}
	return nil
}

func (p *AttributesChartPanel) renderContent(ctx ViewContext, chartWidth int, selectedIdx int, active bool) string {
	maxItems := 10
	if ctx.ContentWidth < 80 {
		maxItems = 5
	}
	if len(p.data) < maxItems {
		maxItems = len(p.data)
	}

	var lines []string

	maxUniqueCount := 0
	for _, attr := range p.data {
		if attr.UniqueValueCount > maxUniqueCount {
			maxUniqueCount = attr.UniqueValueCount
		}
	}

	countFieldWidth := len(fmt.Sprintf("%d", maxUniqueCount))
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

		filled := int((float64(entry.UniqueValueCount) / float64(maxUniqueCount)) * float64(barWidth))
		if filled == 0 && entry.UniqueValueCount > 0 {
			filled = 1
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		key := entry.Key
		if len(key) > labelWidth {
			key = key[:labelWidth-3] + "..."
		}

		formatStr := fmt.Sprintf("%%2d. %%-%ds %%%dd |%%s|", labelWidth, countFieldWidth)
		line := fmt.Sprintf(formatStr, i+1, key, entry.UniqueValueCount, bar)

		if i == selectedIdx && active {
			line = lipgloss.NewStyle().
				Background(ColorBlue).
				Foreground(ColorBlack).
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
