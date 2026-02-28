package tui

import (
	"time"

	"github.com/tinytelemetry/lotus/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

// Deck is a pluggable dashboard deck.
type Deck interface {
	ID() string
	Title() string
	Refresh(store model.LogQuerier, opts model.QueryOpts) // fetch data on tick
	Render(ctx ViewContext, width, height int, active bool, selIdx int) string
	ContentLines(ctx ViewContext) int
	ItemCount() int
	OnSelect(ctx ViewContext, selIdx int) tea.Cmd // returns nil or ActionMsg
}

// TickableDeck extends Deck with independent tick lifecycle methods.
// Decks implementing this interface get their own tick cycle, pause, and error state.
type TickableDeck interface {
	Deck
	TypeID() string                                               // dedup key (e.g. "words")
	DefaultInterval() time.Duration                               // deck's preferred tick interval
	FetchCmd(store model.LogQuerier, opts model.QueryOpts) tea.Cmd // returns DeckDataMsg
	ApplyData(data interface{}, err error)                        // receive fetched data
}
