package tcpserver

import "testing"

func TestNewServer_DefaultLocalhostAddress(t *testing.T) {
	t.Parallel()

	s := NewServer("")
	if got := s.Addr(); got != "127.0.0.1:4000" {
		t.Fatalf("Addr() = %q, want %q", got, "127.0.0.1:4000")
	}
}

func TestNewServer_UsesConfiguredAddressAndBuffers(t *testing.T) {
	t.Parallel()

	s := NewServer("0.0.0.0:5000", ServerConfig{
		LineChannelSize: 64,
		MaxLineSize:     2048,
	})

	if got := s.Addr(); got != "0.0.0.0:5000" {
		t.Fatalf("Addr() = %q, want %q", got, "0.0.0.0:5000")
	}
	if got := cap(s.lineChan); got != 64 {
		t.Fatalf("line channel cap = %d, want %d", got, 64)
	}
	if got := s.maxLineSize; got != 2048 {
		t.Fatalf("max line size = %d, want %d", got, 2048)
	}
}
