package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinytelemetry/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PatternsDeck displays drain3 log patterns.
type PatternsDeck struct {
	drain3Manager *Drain3Manager
	pushModalCmd  tea.Cmd
}

// NewPatternsDeck creates a new patterns deck.
func NewPatternsDeck(drain3 *Drain3Manager, pushModalCmd tea.Cmd) *PatternsDeck {
	return &PatternsDeck{
		drain3Manager: drain3,
		pushModalCmd:  pushModalCmd,
	}
}

func (p *PatternsDeck) ID() string    { return "patterns" }
func (p *PatternsDeck) Title() string { return "Patterns" }

func (p *PatternsDeck) Refresh(_ model.LogQuerier, _ model.QueryOpts) {}

func (p *PatternsDeck) TypeID() string               { return "patterns" }
func (p *PatternsDeck) DefaultInterval() time.Duration { return 2 * time.Second }

// FetchCmd returns a refresh signal (no DB query — patterns come from drain3).
func (p *PatternsDeck) FetchCmd(_ model.LogQuerier, _ model.QueryOpts) tea.Cmd {
	return func() tea.Msg {
		return DeckDataMsg{DeckTypeID: "patterns", Data: nil, Err: nil}
	}
}

// ApplyData is a no-op — patterns are updated as logs arrive via drain3.
func (p *PatternsDeck) ApplyData(_ any, _ error) {}

func (p *PatternsDeck) ContentLines(_ ViewContext) int {
	return 8
}

func (p *PatternsDeck) ItemCount() int {
	return 7
}

func (p *PatternsDeck) Render(ctx ViewContext, width, height int, active bool, _ int) string {
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
	titleText = deckTitleWithBadges(titleText, ctx)
	title := deckTitleStyle.Render(titleText)

	contentLines := height - 3
	if contentLines < 1 {
		contentLines = 1
	}

	var content string
	if p.drain3Manager != nil && patternCount > 0 {
		content = p.renderContent(width, contentLines)
	} else if ctx.DeckLoading {
		content = renderLoadingPlaceholder(width-2, contentLines)
	} else {
		content = helpStyle.Render("Extracting patterns")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

func (p *PatternsDeck) OnSelect(_ ViewContext, _ int) tea.Cmd {
	if p.drain3Manager != nil && p.pushModalCmd != nil {
		return p.pushModalCmd
	}
	return nil
}

func (p *PatternsDeck) renderContent(deckWidth int, availableLines int) string {
	if p.drain3Manager == nil {
		return helpStyle.Render("Pattern extraction not available")
	}

	displayLines := availableLines
	if displayLines < 1 {
		displayLines = 1
	}

	patterns := p.drain3Manager.GetTopPatterns(displayLines)

	maxCount := 0
	for _, pat := range patterns {
		if pat.Count > maxCount {
			maxCount = pat.Count
		}
	}

	templateWidth := deckWidth - 26
	if templateWidth < 20 {
		templateWidth = 20
	}

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
