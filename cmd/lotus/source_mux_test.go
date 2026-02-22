package main

import (
	"context"
	"testing"
	"time"
)

type fakeSource struct {
	name    string
	lines   chan string
	stopped chan struct{}
}

func newFakeSource(name string, buffer int) *fakeSource {
	return &fakeSource{
		name:    name,
		lines:   make(chan string, buffer),
		stopped: make(chan struct{}),
	}
}

func (s *fakeSource) Lines() <-chan string { return s.lines }
func (s *fakeSource) Name() string         { return s.name }

func (s *fakeSource) Stop() {
	select {
	case <-s.stopped:
		return
	default:
		close(s.stopped)
		close(s.lines)
	}
}

func TestSourceMultiplexer_ForwardsFromAllSources(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := newFakeSource("a", 2)
	b := newFakeSource("b", 2)

	mux := NewSourceMultiplexer(ctx, []NamedLogSource{a, b}, 16)
	mux.Start()
	defer mux.Stop()

	a.lines <- "alpha"
	b.lines <- "beta"
	a.Stop()
	b.Stop()

	got := map[string]bool{}
	timeout := time.After(2 * time.Second)
	for len(got) < 2 {
		select {
		case line, ok := <-mux.Lines():
			if !ok {
				t.Fatalf("multiplexer closed before receiving expected lines: %+v", got)
			}
			got[line] = true
		case <-timeout:
			t.Fatalf("timed out waiting for multiplexed lines: %+v", got)
		}
	}

	if !got["alpha"] || !got["beta"] {
		t.Fatalf("missing expected lines: %+v", got)
	}
}

func TestSourceMultiplexer_StopInvokesSourceStop(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	src := newFakeSource("x", 1)
	mux := NewSourceMultiplexer(ctx, []NamedLogSource{src}, 8)
	mux.Start()

	mux.Stop()

	select {
	case <-src.stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("expected source Stop() to be called")
	}
}
