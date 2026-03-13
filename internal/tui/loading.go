package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// renderLoadingPlaceholder renders a loading indicator using a model-driven
// frame index. The frame MUST come from model state (spinnerFrame), not from
// time.Now(), so that View() remains a pure function of the model.
func renderLoadingPlaceholder(width, height, frame int) string {
	idx := frame % len(spinnerFrames)
	if idx < 0 {
		idx = 0
	}

	loadingStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	text := loadingStyle.Render(spinnerFrames[idx] + " Loading...")

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, text)
}

// SpinnerTickMsg triggers a re-render for loading spinners.
type SpinnerTickMsg struct{}

const spinnerInterval = 300 * time.Millisecond

// handleSpinnerTick advances the spinner frame and re-schedules while loading.
func (m *DashboardModel) handleSpinnerTick() (tea.Model, tea.Cmd) {
	m.spinnerFrame++
	if m.anyDeckLoading() {
		return m, tea.Tick(spinnerInterval, func(_ time.Time) tea.Msg {
			return SpinnerTickMsg{}
		})
	}
	return m, nil
}

// anyDeckLoading returns true if any deck has a fetch in flight.
func (m *DashboardModel) anyDeckLoading() bool {
	for _, state := range m.deckStates {
		if state.FetchInFlight {
			return true
		}
	}
	return false
}

// startSpinnerIfNeeded schedules a spinner tick if any deck is loading.
func (m *DashboardModel) startSpinnerIfNeeded() tea.Cmd {
	if m.anyDeckLoading() {
		return tea.Tick(spinnerInterval, func(_ time.Time) tea.Msg {
			return SpinnerTickMsg{}
		})
	}
	return nil
}
