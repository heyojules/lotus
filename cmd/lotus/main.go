package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/control-theory/lotus/internal/socketrpc"

	"github.com/spf13/viper"
)

// Build variables - set by ldflags during build.
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	goVersion = "unknown"
)

// GetVersionInfo returns the current version and commit information.
func GetVersionInfo() (string, string) {
	return version, commit
}

func main() {
	var configPath string
	var showVersion bool

	flag.StringVar(&configPath, "config", "", "config file (default is $HOME/.config/lotus/config.yml)")
	flag.BoolVar(&showVersion, "version", false, "print version information")
	flag.Parse()

	if showVersion {
		fmt.Printf("Lotus - Log Ingestion Service\n")
		fmt.Printf("  Version:    %s\n", version)
		fmt.Printf("  Commit:     %s\n", commit)
		fmt.Printf("  Built:      %s\n", buildTime)
		fmt.Printf("  Go version: %s\n", goVersion)
		return
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := runServer(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig(configPath string) (appConfig, error) {
	var cfg appConfig

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, fmt.Errorf("finding home directory: %w", err)
	}

	defaultDBPath := filepath.Join(home, ".local", "share", "lotus", "lotus.duckdb")

	v := viper.New()
	v.SetEnvPrefix("LOTUS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	v.SetDefault("update-interval", defaultUpdateInterval)
	v.SetDefault("log-buffer", defaultLogBuffer)
	v.SetDefault("test-mode", false)
	v.SetDefault("tcp-enabled", true)
	v.SetDefault("tcp-port", defaultTCPPort)
	v.SetDefault("mux-buffer-size", defaultMuxBufferSize)
	v.SetDefault("db-path", defaultDBPath)
	v.SetDefault("skin", defaultSkin)
	v.SetDefault("disable-version-check", false)
	v.SetDefault("reverse-scroll-wheel", false)
	v.SetDefault("use-log-time", false)
	v.SetDefault("api-enabled", true)
	v.SetDefault("api-port", defaultAPIPort)
	v.SetDefault("query-timeout", defaultQueryTimeout)
	v.SetDefault("max-concurrent-queries", defaultMaxConcurrentReads)
	v.SetDefault("insert-batch-size", defaultInsertBatchSize)
	v.SetDefault("insert-flush-interval", defaultInsertFlushInterval)
	v.SetDefault("insert-flush-queue-size", defaultInsertFlushQueue)
	v.SetDefault("socket-path", socketrpc.DefaultSocketPath())
	v.SetDefault("log-retention", defaultLogRetention)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		defaultConfigPath := filepath.Join(home, ".config", "lotus", "config.yml")
		v.SetConfigFile(defaultConfigPath)
	}

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFound) && !os.IsNotExist(err) {
			return cfg, err
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}
	cfg.ConfigPath = v.ConfigFileUsed()
	if cfg.TCPPort <= 0 || cfg.TCPPort > 65535 {
		return cfg, fmt.Errorf("invalid tcp-port: %d", cfg.TCPPort)
	}
	if cfg.APIPort <= 0 || cfg.APIPort > 65535 {
		return cfg, fmt.Errorf("invalid api-port: %d", cfg.APIPort)
	}

	// Expand ~ in db-path
	if strings.HasPrefix(cfg.DBPath, "~/") {
		cfg.DBPath = filepath.Join(home, cfg.DBPath[2:])
	}

	if cfg.TCPAddr == "" {
		cfg.TCPAddr = net.JoinHostPort(defaultBindHost, strconv.Itoa(cfg.TCPPort))
	}
	if cfg.APIAddr == "" {
		cfg.APIAddr = net.JoinHostPort(defaultBindHost, strconv.Itoa(cfg.APIPort))
	}

	return cfg, nil
}
