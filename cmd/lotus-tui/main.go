package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tinytelemetry/lotus/internal/socketrpc"
	"github.com/tinytelemetry/lotus/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	goVersion = "unknown"
)

func main() {
	var configPath string
	var socketPath string
	var showVersion bool

	flag.StringVar(&configPath, "config", "", "config file (default is $HOME/.config/lotus/config.yml)")
	flag.StringVar(&socketPath, "socket", "", "override socket path to connect to lotus service")
	flag.BoolVar(&showVersion, "version", false, "print version information")
	flag.Parse()

	if showVersion {
		fmt.Printf("Lotus CLI - Dashboard Client\n")
		fmt.Printf("  Version:    %s\n", version)
		fmt.Printf("  Commit:     %s\n", commit)
		fmt.Printf("  Built:      %s\n", buildTime)
		fmt.Printf("  Go version: %s\n", goVersion)
		return
	}

	cfg, err := loadCLIConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if socketPath != "" {
		cfg.SocketPath = socketPath
	}

	if err := runTUI(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runTUI(cfg cliConfig) error {
	configDir := os.Getenv("HOME") + "/.config/lotus"
	if err := tui.InitializeSkin(cfg.Skin, configDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load skin '%s': %v (using default)\n", cfg.Skin, err)
	}

	client, err := socketrpc.Dial(cfg.SocketPath)
	if err != nil {
		return fmt.Errorf("cannot connect to lotus service at %s: %w\nIs the lotus service running? Start it with: lotus", cfg.SocketPath, err)
	}
	defer client.Close()

	dashboard := tui.NewDashboardModel(cfg.LogBuffer, cfg.UpdateInterval, cfg.ReverseScrollWheel, cfg.UseLogTime, client, "Socket")
	dashPage := tui.NewDashboardPage(dashboard)
	app := tui.NewApp(dashPage)

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		if strings.Contains(err.Error(), "TTY") || strings.Contains(err.Error(), "/dev/tty") {
			return fmt.Errorf("TUI requires a real terminal")
		}
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
