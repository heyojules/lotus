package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Chart calculation helpers

// calculateRequiredChartsHeight calculates how much vertical space the charts need
func (m *DashboardModel) calculateRequiredChartsHeight() int {
	if len(m.panels) == 0 {
		return 3
	}

	totalRequired := m.chartAreaHeight()
	return totalRequired
}

func (m *DashboardModel) chartColumnCount() int {
	if len(m.panels) <= 1 {
		return 1
	}
	return 2
}

func (m *DashboardModel) chartPanelHeight(idx int) int {
	h := m.panels[idx].ContentLines(m.viewContext()) + 3
	if h < 4 {
		return 4
	}
	return h
}

func (m *DashboardModel) chartRowHeights() []int {
	if len(m.panels) == 0 {
		return nil
	}

	cols := m.chartColumnCount()
	rows := (len(m.panels) + cols - 1) / cols
	heights := make([]int, rows)

	for row := 0; row < rows; row++ {
		rowHeight := 4
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(m.panels) {
				break
			}
			rowHeight = max(rowHeight, m.chartPanelHeight(idx))
		}
		heights[row] = rowHeight
	}

	return heights
}

func (m *DashboardModel) chartAreaHeight() int {
	total := 0
	for _, h := range m.chartRowHeights() {
		total += h
	}
	return total
}

func (m *DashboardModel) chartRowHeightsFor(height int) []int {
	required := m.chartRowHeights()
	if len(required) == 0 {
		return nil
	}

	totalReq := 0
	for _, h := range required {
		totalReq += h
	}

	if totalReq <= 0 {
		return required
	}

	if totalReq <= height {
		return required
	}

	scaled := make([]int, len(required))
	remaining := height
	for i, h := range required {
		if i == len(required)-1 {
			scaled[i] = max(3, remaining)
			break
		}
		value := int(float64(h) * float64(height) / float64(totalReq))
		if value < 3 {
			value = 3
		}
		scaled[i] = value
		remaining -= value
	}

	if remaining > 0 {
		scaled[len(scaled)-1] += remaining
	}

	return scaled
}

func (m *DashboardModel) chartPanelAt(contentWidth int, chartHeight int, x int, y int) (int, bool) {
	if len(m.panels) == 0 || x < 0 || y < 0 {
		return 0, false
	}

	cols := m.chartColumnCount()
	if cols <= 0 {
		return 0, false
	}

	panelWidth := contentWidth
	colGap := 0
	if cols > 1 {
		colGap = 1
		panelWidth = max(1, (contentWidth-colGap)/cols)
	}

	rowHeights := m.chartRowHeightsFor(chartHeight)
	rowY := 0
	for row, rowHeight := range rowHeights {
		if y < rowY+rowHeight {
			col := 0
			if cols > 1 {
				stride := panelWidth + colGap
				col = x / stride
				if col >= cols {
					col = cols - 1
				}
			}
			idx := row*cols + col
			if idx >= len(m.panels) {
				return 0, false
			}
			return idx, true
		}
		rowY += rowHeight
	}

	return 0, false
}

// Chart rendering functions

// renderChartsGrid renders a two-column chart grid (single-column when only one panel).
func (m *DashboardModel) renderChartsGrid(width int, height int) string {
	if width < 20 {
		return "Terminal too narrow"
	}

	if len(m.panels) == 0 {
		return "No panels registered"
	}

	cols := m.chartColumnCount()
	rowHeights := m.chartRowHeightsFor(height)
	rows := len(rowHeights)

	// Each chart panel adds 2 chars for borders (left+right) on top of its Width.
	// Account for this so the total rendered row fits within the available width.
	borderWidth := 2
	chartWidth := width - borderWidth
	colGap := 0
	if cols > 1 {
		colGap = 1
		chartWidth = (width - colGap - cols*borderWidth) / cols
		if chartWidth < 25 {
			chartWidth = 25
		}
	}

	blankPanel := func(panelHeight int) string {
		return lipgloss.NewStyle().
			Width(chartWidth).
			Height(panelHeight).
			Render("")
	}

	ctx := m.viewContext()
	renderPanel := func(idx int, h int) string {
		active := m.activeSection == SectionCharts && m.activePanelIdx == idx
		selIdx := m.panelSelIdx[idx]
		return m.panels[idx].Render(ctx, chartWidth, h, active, selIdx)
	}

	renderedRows := make([]string, 0, rows)
	for row := 0; row < rows; row++ {
		panelHeight := rowHeights[row]
		rowPanels := make([]string, 0, cols)

		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(m.panels) {
				if cols > 1 {
					rowPanels = append(rowPanels, blankPanel(panelHeight))
				}
				continue
			}
			rowPanels = append(rowPanels, renderPanel(idx, panelHeight))
		}

		rowView := rowPanels[0]
		if len(rowPanels) > 1 {
			withGaps := make([]string, 0, len(rowPanels)*2-1)
			for i, panel := range rowPanels {
				if i > 0 {
					withGaps = append(withGaps, " ")
				}
				withGaps = append(withGaps, panel)
			}
			rowView = lipgloss.JoinHorizontal(lipgloss.Top, withGaps...)
		}
		renderedRows = append(renderedRows, rowView)
	}

	result := lipgloss.JoinVertical(lipgloss.Left, renderedRows...)

	constrainedStyle := lipgloss.NewStyle().
		MaxHeight(height).
		Width(width)

	return constrainedStyle.Render(result)
}
