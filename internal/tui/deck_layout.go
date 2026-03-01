package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Chart calculation helpers

// calculateRequiredDecksHeight calculates how much vertical space the charts need
func (m *DashboardModel) calculateRequiredDecksHeight() int {
	if len(m.decks) == 0 {
		return 3
	}

	totalRequired := m.deckAreaHeight()
	return totalRequired
}

func (m *DashboardModel) deckColumnCount() int {
	if len(m.decks) <= 1 {
		return 1
	}
	return 2
}

func (m *DashboardModel) deckHeight(idx int) int {
	h := m.decks[idx].ContentLines(m.viewContext()) + 3
	if h < 4 {
		return 4
	}
	return h
}

func (m *DashboardModel) deckRowHeights() []int {
	if len(m.decks) == 0 {
		return nil
	}

	cols := m.deckColumnCount()
	rows := (len(m.decks) + cols - 1) / cols
	heights := make([]int, rows)

	for row := 0; row < rows; row++ {
		rowHeight := 4
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(m.decks) {
				break
			}
			rowHeight = max(rowHeight, m.deckHeight(idx))
		}
		heights[row] = rowHeight
	}

	return heights
}

func (m *DashboardModel) deckAreaHeight() int {
	total := 0
	for _, h := range m.deckRowHeights() {
		total += h
	}
	return total
}

func (m *DashboardModel) deckRowHeightsFor(height int) []int {
	required := m.deckRowHeights()
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

	// Distribute height equally across rows.
	rows := len(required)
	perRow := height / rows
	if perRow < 3 {
		perRow = 3
	}

	scaled := make([]int, rows)
	for i := range scaled {
		scaled[i] = perRow
	}
	// Give the last row any remaining pixels.
	scaled[rows-1] = height - perRow*(rows-1)
	if scaled[rows-1] < 3 {
		scaled[rows-1] = 3
	}

	return scaled
}

func (m *DashboardModel) deckAt(contentWidth int, chartHeight int, x int, y int) (int, bool) {
	if len(m.decks) == 0 || x < 0 || y < 0 {
		return 0, false
	}

	cols := m.deckColumnCount()
	if cols <= 0 {
		return 0, false
	}

	deckWidth := contentWidth
	colGap := 0
	if cols > 1 {
		colGap = 1
		deckWidth = max(1, (contentWidth-colGap)/cols)
	}

	rowHeights := m.deckRowHeightsFor(chartHeight)
	rowY := 0
	for row, rowHeight := range rowHeights {
		if y < rowY+rowHeight {
			col := 0
			if cols > 1 {
				stride := deckWidth + colGap
				col = x / stride
				if col >= cols {
					col = cols - 1
				}
			}
			idx := row*cols + col
			if idx >= len(m.decks) {
				return 0, false
			}
			return idx, true
		}
		rowY += rowHeight
	}

	return 0, false
}

// deckTitleWithBadges appends pause/error badges to a deck title based on ViewContext.
func deckTitleWithBadges(title string, ctx ViewContext) string {
	if ctx.DeckPaused {
		title += " ⏸"
	}
	if ctx.DeckLastError != "" {
		title += " ⚠"
	}
	return title
}

// Deck rendering functions

// renderDecksGrid renders a two-column deck grid (single-column when only one panel).
func (m *DashboardModel) renderDecksGrid(width int, height int) string {
	if width < 20 {
		return "Terminal too narrow"
	}

	if len(m.decks) == 0 {
		return "No decks registered"
	}

	cols := m.deckColumnCount()
	rowHeights := m.deckRowHeightsFor(height)
	rows := len(rowHeights)

	// Each deck adds 2 chars for borders (left+right) on top of its Width.
	// Account for this so the total rendered row fits within the available width.
	borderWidth := 2
	deckWidth := width - borderWidth
	colGap := 0
	if cols > 1 {
		colGap = 1
		deckWidth = (width - colGap - cols*borderWidth) / cols
		if deckWidth < 25 {
			deckWidth = 25
		}
	}

	blankDeck := func(deckHeight int) string {
		return lipgloss.NewStyle().
			Width(deckWidth).
			Height(deckHeight).
			Render("")
	}

	baseCtx := m.viewContext()
	renderDeck := func(idx int, h int) string {
		active := m.activeSection == SectionDecks && m.activeDeckIdx == idx
		selIdx := m.deckSelIdx[idx]
		ctx := baseCtx
		// Inject per-deck pause/error state into ViewContext.
		if tp, ok := m.decks[idx].(TickableDeck); ok {
			if state, exists := m.deckStates[tp.TypeID()]; exists {
				ctx.DeckPaused = state.Paused
				ctx.DeckLastError = state.LastError
				ctx.DeckLoading = state.FetchInFlight
			}
		}
		return m.decks[idx].Render(ctx, deckWidth, h, active, selIdx)
	}

	renderedRows := make([]string, 0, rows)
	for row := 0; row < rows; row++ {
		deckHeight := rowHeights[row]
		rowDecks := make([]string, 0, cols)

		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(m.decks) {
				if cols > 1 {
					rowDecks = append(rowDecks, blankDeck(deckHeight))
				}
				continue
			}
			rowDecks = append(rowDecks, renderDeck(idx, deckHeight))
		}

		rowView := rowDecks[0]
		if len(rowDecks) > 1 {
			withGaps := make([]string, 0, len(rowDecks)*2-1)
			for i, panel := range rowDecks {
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
		Height(height).
		MaxHeight(height).
		Width(width)

	return constrainedStyle.Render(result)
}
