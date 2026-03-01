package logsource

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestStdinSourceStopClosesLines(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = w.Close() }()

	src := newStdinSourceWithReader(context.Background(), r)
	src.Stop()

	select {
	case _, ok := <-src.Lines():
		if ok {
			t.Fatal("expected lines channel to be closed after Stop")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for lines channel to close")
	}
}

func TestStdinSourceStopIsIdempotent(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = w.Close() }()

	src := newStdinSourceWithReader(context.Background(), r)
	src.Stop()
	src.Stop()
}
