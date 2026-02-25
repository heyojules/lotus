package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tinytelemetry/lotus/internal/logsource"
	"github.com/tinytelemetry/lotus/internal/tcpserver"
)

// NamedLogSource aliases the shared source abstraction to keep app-layer APIs explicit.
type NamedLogSource = logsource.LogSource

// InputSourcePlugin is a small plugin primitive for wiring log inputs.
type InputSourcePlugin interface {
	Name() string
	Enabled() bool
	Build(ctx context.Context) (NamedLogSource, error)
}

// InputPluginConfig defines runtime input selection.
type InputPluginConfig struct {
	TCPEnabled bool
	TCPAddr    string
}

func buildInputPlugins(cfg InputPluginConfig) []InputSourcePlugin {
	plugins := make([]InputSourcePlugin, 0, 2)
	plugins = append(plugins, tcpInputPlugin{
		addr:    cfg.TCPAddr,
		enabled: cfg.TCPEnabled,
	})
	plugins = append(plugins, stdinInputPlugin{})
	return plugins
}

type tcpInputPlugin struct {
	addr    string
	enabled bool
}

func (p tcpInputPlugin) Name() string { return "tcp" }

func (p tcpInputPlugin) Enabled() bool { return p.enabled }

func (p tcpInputPlugin) Build(_ context.Context) (NamedLogSource, error) {
	server := tcpserver.NewServer(p.addr)
	if err := server.Start(); err != nil {
		return nil, fmt.Errorf("start tcp server: %w", err)
	}
	return logsource.NewTCPSource(server), nil
}

type stdinInputPlugin struct{}

func (p stdinInputPlugin) Name() string { return "stdin" }

func (p stdinInputPlugin) Enabled() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func (p stdinInputPlugin) Build(ctx context.Context) (NamedLogSource, error) {
	return logsource.NewStdinSource(ctx), nil
}
