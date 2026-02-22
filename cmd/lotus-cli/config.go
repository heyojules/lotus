package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/control-theory/lotus/internal/model"
	"github.com/control-theory/lotus/internal/socketrpc"
	"github.com/spf13/viper"
)

const (
	defaultUpdateInterval = model.DefaultUpdateInterval
	defaultLogBuffer      = model.DefaultLogBuffer
	defaultSkin           = model.DefaultSkin
)

// cliConfig holds only TUI-relevant configuration.
type cliConfig struct {
	UpdateInterval     time.Duration `mapstructure:"update-interval"`
	LogBuffer          int           `mapstructure:"log-buffer"`
	Skin               string        `mapstructure:"skin"`
	ReverseScrollWheel bool          `mapstructure:"reverse-scroll-wheel"`
	UseLogTime         bool          `mapstructure:"use-log-time"`
	SocketPath         string        `mapstructure:"socket-path"`
}

func loadCLIConfig(configPath string) (cliConfig, error) {
	var cfg cliConfig

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, fmt.Errorf("finding home directory: %w", err)
	}

	v := viper.New()
	v.SetEnvPrefix("LOTUS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	v.SetDefault("update-interval", defaultUpdateInterval)
	v.SetDefault("log-buffer", defaultLogBuffer)
	v.SetDefault("skin", defaultSkin)
	v.SetDefault("reverse-scroll-wheel", false)
	v.SetDefault("use-log-time", false)
	v.SetDefault("socket-path", socketrpc.DefaultSocketPath())

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigFile(filepath.Join(home, ".config", "lotus", "config.yml"))
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

	return cfg, nil
}
