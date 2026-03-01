package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinytelemetry/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AttributesDeck displays the most frequent attribute keys.
type AttributesDeck struct {
	store          model.LogQuerier
	formatModal    func(entry *AttributeEntry, maxWidth int) string
	pushContentCmd func(content string) tea.Cmd
	data           []AttributeEntry
}

// NewAttributesDeck creates a new attributes deck.
func NewAttributesDeck(store model.LogQuerier, formatModal func(entry *AttributeEntry, maxWidth int) string, pushContentCmd func(content string) tea.Cmd) *AttributesDeck {
	return &AttributesDeck{
		store:          store,
		formatModal:    formatModal,
		pushContentCmd: pushContentCmd,
	}
}

func (p *AttributesDeck) ID() string    { return "attributes" }
func (p *AttributesDeck) Title() string { return "Attrs" }

func (p *AttributesDeck) Refresh(_ model.LogQuerier, _ model.QueryOpts) {}

func (p *AttributesDeck) TypeID() string               { return "attributes" }
func (p *AttributesDeck) DefaultInterval() time.Duration { return 2 * time.Second }

func (p *AttributesDeck) FetchCmd(store model.LogQuerier, opts model.QueryOpts) tea.Cmd {
	return func() tea.Msg {
		attrKeys, err := store.TopAttributeKeys(50, opts)
		var entries []AttributeEntry
		if err == nil {
			entries = make([]AttributeEntry, len(attrKeys))
			for i, ak := range attrKeys {
				entries[i] = AttributeEntry{
					Key:              ak.Key,
					UniqueValueCount: ak.UniqueValues,
					TotalCount:       ak.TotalCount,
				}
			}
		}
		return DeckDataMsg{DeckTypeID: "attributes", Data: entries, Err: err}
	}
}

func (p *AttributesDeck) ApplyData(data any, err error) {
	if err != nil {
		return
	}
	if entries, ok := data.([]AttributeEntry); ok {
		p.data = append([]AttributeEntry(nil), entries...)
	}
}

func (p *AttributesDeck) ContentLines(ctx ViewContext) int {
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

func (p *AttributesDeck) ItemCount() int {
	return min(len(p.data), 10)
}

func (p *AttributesDeck) Render(ctx ViewContext, width, height int, active bool, selIdx int) string {
	style := sectionStyle.Width(width).Height(height)
	if active {
		style = activeSectionStyle.Width(width).Height(height)
	}

	title := deckTitleStyle.Render(deckTitleWithBadges("Top Attributes", ctx))

	contentLines := height - 3
	if contentLines < 1 {
		contentLines = 1
	}

	var content string
	if len(p.data) > 0 {
		content = p.renderContent(ctx, width, contentLines, selIdx, active)
	} else if ctx.DeckLoading {
		content = renderLoadingPlaceholder(width-2, contentLines)
	} else {
		content = helpStyle.Render("No data available")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

func (p *AttributesDeck) OnSelect(ctx ViewContext, selIdx int) tea.Cmd {
	if selIdx < len(p.data) {
		if p.formatModal == nil || p.pushContentCmd == nil {
			return nil
		}
		entry := p.data[selIdx]
		// Fetch heavy value distribution on demand to avoid N+1 queries on each tick.
		if p.store != nil {
			if values, err := p.store.AttributeKeyValues(entry.Key, 100); err == nil {
				entry.Values = values
			}
		}
		contentWidth := ctx.ContentWidth - 16
		if contentWidth < 60 {
			contentWidth = 60
		}
		content := p.formatModal(&entry, contentWidth)
		return p.pushContentCmd(content)
	}
	return nil
}

func (p *AttributesDeck) renderContent(ctx ViewContext, deckWidth int, availableLines int, selectedIdx int, active bool) string {
	maxItems := min(len(p.data), availableLines)
	if maxItems < 1 {
		maxItems = 1
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

	availableWidth := deckWidth - 2
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
