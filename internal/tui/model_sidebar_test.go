package tui

import (
	"testing"
	"time"
)

func TestNewDashboardModel_DefaultSidebarAndSection(t *testing.T) {
	t.Parallel()

	model := NewDashboardModel(1000, time.Second, false, false, nil, "")
	if !model.sidebarVisible {
		t.Fatal("expected sidebar to be visible by default")
	}
	if model.activeSection != SectionCharts {
		t.Fatalf("expected initial active section to be charts, got %v", model.activeSection)
	}
}
