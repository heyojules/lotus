package tui

import (
	"testing"
)

func TestDrain3Manager_AddLogMessage(t *testing.T) {
	t.Parallel()
	dm := NewDrain3Manager()

	dm.AddLogMessage("Connection refused from 192.168.1.1")
	dm.AddLogMessage("Connection refused from 10.0.0.1")
	dm.AddLogMessage("Connection refused from 172.16.0.1")

	patterns := dm.GetTopPatterns(10)
	if len(patterns) == 0 {
		t.Fatal("expected at least one pattern")
	}

	_, total := dm.GetStats()
	if total != 3 {
		t.Errorf("total logs = %d, want 3", total)
	}
}

func TestDrain3Manager_EmptyMessage(t *testing.T) {
	t.Parallel()
	dm := NewDrain3Manager()

	dm.AddLogMessage("")
	dm.AddLogMessage("   ")

	_, total := dm.GetStats()
	if total != 0 {
		t.Errorf("total logs = %d, want 0 (empty messages should be skipped)", total)
	}
}

func TestDrain3Manager_Reset(t *testing.T) {
	t.Parallel()
	dm := NewDrain3Manager()

	dm.AddLogMessage("test message")
	dm.Reset()

	patterns := dm.GetTopPatterns(10)
	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns after reset, got %d", len(patterns))
	}

	_, total := dm.GetStats()
	if total != 0 {
		t.Errorf("total logs = %d after reset, want 0", total)
	}
}

func TestDrain3Manager_GetTopPatterns_Sorted(t *testing.T) {
	t.Parallel()
	dm := NewDrain3Manager()

	// Add different patterns with different frequencies
	for i := 0; i < 10; i++ {
		dm.AddLogMessage("frequent pattern message here")
	}
	for i := 0; i < 3; i++ {
		dm.AddLogMessage("rare pattern something")
	}

	patterns := dm.GetTopPatterns(10)
	if len(patterns) < 2 {
		t.Skipf("drain3 merged patterns, got %d (expected 2+)", len(patterns))
	}

	// Patterns should be sorted by count descending
	for i := 1; i < len(patterns); i++ {
		if patterns[i].Count > patterns[i-1].Count {
			t.Errorf("patterns not sorted: index %d count %d > index %d count %d",
				i, patterns[i].Count, i-1, patterns[i-1].Count)
		}
	}
}

func TestDrain3Manager_GetTopPatterns_Limit(t *testing.T) {
	t.Parallel()
	dm := NewDrain3Manager()

	// Add many distinct patterns
	for i := 0; i < 100; i++ {
		dm.AddLogMessage("unique message number something different " + string(rune('A'+i%26)))
	}

	patterns := dm.GetTopPatterns(3)
	if len(patterns) > 3 {
		t.Errorf("expected at most 3 patterns, got %d", len(patterns))
	}
}

func TestDrain3Manager_Percentages(t *testing.T) {
	t.Parallel()
	dm := NewDrain3Manager()

	for i := 0; i < 10; i++ {
		dm.AddLogMessage("test message")
	}

	patterns := dm.GetTopPatterns(10)
	if len(patterns) == 0 {
		t.Fatal("expected patterns")
	}

	totalPct := 0.0
	for _, p := range patterns {
		totalPct += p.Percentage
	}
	// Total percentage should be approximately 100
	if totalPct < 99.0 || totalPct > 101.0 {
		t.Errorf("total percentage = %.1f, want ~100", totalPct)
	}
}
