package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// renderLoadingPlaceholder renders an animated loading indicator.
// The frame is selected based on the current time so it animates on re-render.
func renderLoadingPlaceholder(width, height int) string {
	frame := spinnerFrames[time.Now().UnixMilli()/120%int64(len(spinnerFrames))]

	loadingStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	text := loadingStyle.Render(frame + " Loading...")

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, text)
}

// SpinnerTickMsg triggers a re-render for loading spinners.
type SpinnerTickMsg struct{}

// handleSpinnerTick re-schedules spinner ticks while any deck is loading.
func (m *DashboardModel) handleSpinnerTick() (tea.Model, tea.Cmd) {
	if m.anyDeckLoading() {
		return m, tea.Tick(120*time.Millisecond, func(_ time.Time) tea.Msg {
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
		return tea.Tick(120*time.Millisecond, func(_ time.Time) tea.Msg {
			return SpinnerTickMsg{}
		})
	}
	return nil
}
