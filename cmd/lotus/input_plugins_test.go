package main

import "testing"

func TestBuildInputPlugins_RegistersPrimitives(t *testing.T) {
	t.Parallel()

	plugins := buildInputPlugins(InputPluginConfig{
		TCPEnabled: true,
		TCPAddr:    "127.0.0.1:4000",
	})

	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if plugins[0].Name() != "tcp" {
		t.Fatalf("plugins[0] name = %q, want %q", plugins[0].Name(), "tcp")
	}
	if plugins[1].Name() != "stdin" {
		t.Fatalf("plugins[1] name = %q, want %q", plugins[1].Name(), "stdin")
	}
	if !plugins[0].Enabled() {
		t.Fatal("expected tcp plugin to be enabled when TCPEnabled=true")
	}
}

func TestBuildInputPlugins_TCPDisabled(t *testing.T) {
	t.Parallel()

	plugins := buildInputPlugins(InputPluginConfig{
		TCPEnabled: false,
		TCPAddr:    "127.0.0.1:4000",
	})

	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if plugins[0].Enabled() {
		t.Fatal("expected tcp plugin to be disabled when TCPEnabled=false")
	}
}
