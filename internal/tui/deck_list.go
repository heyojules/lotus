package tui

import (
	"strings"

	"github.com/tinytelemetry/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListDeck wraps the log scroll as a proper Deck so the List view
// renders through the standard renderDecksGrid path.
type ListDeck struct {
	m *DashboardModel
}

// NewListDeck creates a ListDeck backed by the given model.
func NewListDeck(m *DashboardModel) *ListDeck {
	return &ListDeck{m: m}
}

func (d *ListDeck) ID() string    { return "list" }
func (d *ListDeck) Title() string { return "Log List" }

func (d *ListDeck) Refresh(_ model.LogQuerier, _ model.QueryOpts) {}

// ContentLines returns a large value so the grid scaler gives this deck
// all available height (it is the only deck in the List view).
func (d *ListDeck) ContentLines(_ ViewContext) int { return 100 }

// ItemCount returns the number of log entries for deck selection navigation.
func (d *ListDeck) ItemCount() int { return len(d.m.logEntries) }

func (d *ListDeck) Render(ctx ViewContext, width, height int, active bool, selIdx int) string {
	style := sectionStyle.Width(width).Height(height)
	if active {
		style = activeSectionStyle.Width(width).Height(height)
	}

	title := deckTitleStyle.Render("Log List")

	// Available content lines = height minus border (2) and title (1).
	contentHeight := height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Sync deck selection index → model's selectedLogIndex when active.
	if active {
		d.m.selectedLogIndex = selIdx
	}

	// Temporarily set activeSection to SectionLogs so renderLogScrollContent
	// shows focus-lock indicator, centers selection, and highlights correctly.
	origSection := d.m.activeSection
	if active {
		d.m.activeSection = SectionLogs
	}

	logLines := d.m.renderLogScrollContent(contentHeight, width-2) // -2 for borders

	d.m.activeSection = origSection

	content := strings.Join(logLines, "\n")

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

// OnSelect opens the detail modal for the selected log entry.
func (d *ListDeck) OnSelect(_ ViewContext, selIdx int) tea.Cmd {
	if selIdx >= 0 && selIdx < len(d.m.logEntries) {
		entry := d.m.logEntries[selIdx]
		modal := NewDetailModal(d.m, &entry)
		return actionMsg(ActionMsg{Action: ActionPushModal, Payload: modal})
	}
	return nil
}
