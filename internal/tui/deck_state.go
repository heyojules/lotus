package tui

import "time"

// DeckTypeState tracks per-TypeID tick/pause/error state.
type DeckTypeState struct {
	TypeID          string
	Interval        time.Duration
	Paused          bool
	FetchInFlight   bool
	LastError       string
	LastErrorAt     time.Time
	LastTickOK      bool
	LastTickAt      time.Time
	ConsecutiveErrs int
}
