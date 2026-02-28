package tui

import (
	"testing"
	"time"
)

func TestNewDashboardModel_DefaultPages(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")

	if got := len(m.pages); got < 1 {
		t.Fatalf("default pages = %d, want >= 1", got)
	}
	if got := m.currentPageTitle(); got == "" {
		t.Fatal("current page title is empty")
	}
	if got := m.currentViewTitle(); got == "" {
		t.Fatal("current view title is empty")
	}
	if got := len(m.decks); got == 0 {
		t.Fatal("active view has no decks")
	}
}

func TestViewSwitch_PreservesDeckState(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")
	pg := m.activePage()
	if pg == nil || len(pg.Views) < 2 {
		t.Skip("need at least two views in the active page")
	}

	m.activeDeckIdx = min(2, len(m.decks)-1)
	m.deckSelIdx[m.activeDeckIdx] = 1

	m.nextView()
	m.prevView()

	if got := m.activeDeckIdx; got != min(2, len(m.decks)-1) {
		t.Fatalf("active deck idx = %d, want preserved", got)
	}
	if got := m.deckSelIdx[m.activeDeckIdx]; got != 1 {
		t.Fatalf("deck selection = %d, want preserved", got)
	}
}

func TestSidebarActivation_SelectsPageAndApp(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")
	m.appList = []string{"api", "worker"}

	// Select "worker" app from sidebar: api + worker.
	m.sidebarCursor = 1
	m.activateSidebarCursor()
	if got := m.selectedApp; got != "worker" {
		t.Fatalf("selected app = %q, want worker", got)
	}

	// Select second page from sidebar: apps come first, then pages.
	m.sidebarCursor = len(m.appList) + 1
	m.activateSidebarCursor()
	if got := m.currentPageTitle(); got != m.pages[1].Title {
		t.Fatalf("active page = %q, want %q", got, m.pages[1].Title)
	}
}

func TestPageSwitch_LoadsCorrectViews(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")
	if len(m.pages) < 2 {
		t.Skip("need at least two pages")
	}

	// Logs page should have 3 views
	logsPage := m.pages[0]
	if got := len(logsPage.Views); got != 3 {
		t.Fatalf("logs page views = %d, want 3", got)
	}

	// Switch to Metrics page (placeholder with 1 view, no decks)
	m.activatePage(1)
	if got := m.currentPageTitle(); got != "Metrics" {
		t.Fatalf("page title = %q, want Metrics", got)
	}
	// Metrics placeholder has no decks
	if got := len(m.decks); got != 0 {
		t.Fatalf("metrics decks = %d, want 0", got)
	}

	// Switch back to Logs
	m.activatePage(0)
	if got := len(m.decks); got == 0 {
		t.Fatal("logs page should have decks after switching back")
	}
}
