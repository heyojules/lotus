package tui

import (
	"testing"
	"time"
)

func TestNewDashboardModel_DefaultDeckPages(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")

	if got := len(m.deckPages); got < 1 {
		t.Fatalf("default deck pages = %d, want >= 1", got)
	}
	if got := m.currentPageTitle(); got == "" {
		t.Fatal("current page title is empty")
	}
	if got := len(m.panels); got == 0 {
		t.Fatal("active page has no panels")
	}
}

func TestPageSwitch_PreservesPanelState(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")
	if len(m.deckPages) < 2 {
		t.Skip("need at least two pages")
	}

	m.activePanelIdx = min(2, len(m.panels)-1)
	m.panelSelIdx[m.activePanelIdx] = 1

	m.nextPage()
	m.prevPage()

	if got := m.activePanelIdx; got != min(2, len(m.panels)-1) {
		t.Fatalf("active panel idx = %d, want preserved", got)
	}
	if got := m.panelSelIdx[m.activePanelIdx]; got != 1 {
		t.Fatalf("panel selection = %d, want preserved", got)
	}
}

func TestSidebarActivation_SelectsPageAndApp(t *testing.T) {
	t.Parallel()

	m := NewDashboardModel(1000, time.Second, false, false, nil, "")
	m.appList = []string{"api", "worker"}

	// Select second page from sidebar.
	m.sidebarCursor = 1
	m.activateSidebarCursor()
	if got := m.currentPageTitle(); got != m.deckPages[1].Title {
		t.Fatalf("active page = %q, want %q", got, m.deckPages[1].Title)
	}

	// Select "worker" app from sidebar: pages + All + api + worker.
	m.sidebarCursor = len(m.deckPages) + 2
	m.activateSidebarCursor()
	if got := m.selectedApp; got != "worker" {
		t.Fatalf("selected app = %q, want worker", got)
	}
}
