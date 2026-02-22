package tui

import tea "github.com/charmbracelet/bubbletea"

// App is the top-level Bubble Tea model that routes between pages.
type App struct {
	pages      map[string]Page
	activePage string
	width      int
	height     int
}

// NewApp creates a new App with the given pages. The first page is the default.
func NewApp(pages ...Page) *App {
	pageMap := make(map[string]Page, len(pages))
	var firstID string
	for i, p := range pages {
		pageMap[p.ID()] = p
		if i == 0 {
			firstID = p.ID()
		}
	}
	return &App{
		pages:      pageMap,
		activePage: firstID,
	}
}

func (a *App) Init() tea.Cmd {
	if p, ok := a.pages[a.activePage]; ok {
		return p.Init()
	}
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Pass WindowSizeMsg to all pages so they can track dimensions.
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		a.width = wsm.Width
		a.height = wsm.Height
	}

	p, ok := a.pages[a.activePage]
	if !ok {
		return a, nil
	}

	cmd, nav := p.Update(msg)

	if nav != nil {
		if _, exists := a.pages[nav.PageID]; exists {
			a.activePage = nav.PageID
			initCmd := a.pages[a.activePage].Init()
			return a, tea.Batch(cmd, initCmd)
		}
	}

	return a, cmd
}

func (a *App) View() string {
	if p, ok := a.pages[a.activePage]; ok {
		return p.View(a.width, a.height)
	}
	return "No active page"
}
