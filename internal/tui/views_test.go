package tui

import (
	"testing"
	"time"
)

func TestNewDashboardModel_DefaultViews(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")

	if got := len(m.views); got < 1 {
		t.Fatalf("default deck pages = %d, want >= 1", got)
	}
	if got := m.currentViewTitle(); got == "" {
		t.Fatal("current page title is empty")
	}
	if got := len(m.decks); got == 0 {
		t.Fatal("active page has no panels")
	}
}

func TestPageSwitch_PreservesPanelState(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")
	if len(m.views) < 2 {
		t.Skip("need at least two pages")
	}

	m.activeDeckIdx = min(2, len(m.decks)-1)
	m.deckSelIdx[m.activeDeckIdx] = 1

	m.nextView()
	m.prevView()

	if got := m.activeDeckIdx; got != min(2, len(m.decks)-1) {
		t.Fatalf("active panel idx = %d, want preserved", got)
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

	// Select second page from sidebar: apps + pages.
	m.sidebarCursor = len(m.appList) + 1
	m.activateSidebarCursor()
	if got := m.currentViewTitle(); got != m.views[1].Title {
		t.Fatalf("active page = %q, want %q", got, m.views[1].Title)
	}
}
