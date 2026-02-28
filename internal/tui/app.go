package tui

import tea "github.com/charmbracelet/bubbletea"

// App is the top-level Bubble Tea model that routes between views.
type App struct {
	views      map[string]View
	activeView string
	width      int
	height     int
}

// NewApp creates a new App with the given views. The first view is the default.
func NewApp(views ...View) *App {
	viewMap := make(map[string]View, len(views))
	var firstID string
	for i, v := range views {
		viewMap[v.ID()] = v
		if i == 0 {
			firstID = v.ID()
		}
	}
	return &App{
		views:      viewMap,
		activeView: firstID,
	}
}

func (a *App) Init() tea.Cmd {
	if v, ok := a.views[a.activeView]; ok {
		return v.Init()
	}
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Pass WindowSizeMsg to all views so they can track dimensions.
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		a.width = wsm.Width
		a.height = wsm.Height
	}

	v, ok := a.views[a.activeView]
	if !ok {
		return a, nil
	}

	cmd, nav := v.Update(msg)

	if nav != nil {
		if _, exists := a.views[nav.ViewID]; exists {
			a.activeView = nav.ViewID
			initCmd := a.views[a.activeView].Init()
			return a, tea.Batch(cmd, initCmd)
		}
	}

	return a, cmd
}

func (a *App) View() string {
	if v, ok := a.views[a.activeView]; ok {
		return v.View(a.width, a.height)
	}
	return "No active view"
}
