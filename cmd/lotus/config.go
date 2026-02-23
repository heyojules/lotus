package main

import (
	"time"

	"github.com/control-theory/lotus/internal/model"
)

const (
	defaultUpdateInterval      = model.DefaultUpdateInterval
	defaultLogBuffer           = model.DefaultLogBuffer
	defaultBindHost            = "127.0.0.1"
	defaultTCPPort             = 4000
	defaultSkin                = model.DefaultSkin
	defaultAPIPort             = 3000
	defaultQueryTimeout        = 30 * time.Second
	defaultInsertBatchSize     = 2000
	defaultInsertFlushInterval = 100 * time.Millisecond
	defaultLogRetention        = 30 // days, 0 = disabled
)

// appConfig is internal runtime configuration.
// It is package-private to keep defaults and shape local to the CLI entrypoint.
type appConfig struct {
	UpdateInterval      time.Duration `mapstructure:"update-interval"`
	LogBuffer           int           `mapstructure:"log-buffer"`
	TestMode            bool          `mapstructure:"test-mode"`
	TCPEnabled          bool          `mapstructure:"tcp-enabled"`
	TCPPort             int           `mapstructure:"tcp-port"`
	TCPAddr             string        `mapstructure:"tcp-addr"`
	DBPath              string        `mapstructure:"db-path"`
	Skin                string        `mapstructure:"skin"`
	DisableVersionCheck bool          `mapstructure:"disable-version-check"`
	ReverseScrollWheel  bool          `mapstructure:"reverse-scroll-wheel"`
	UseLogTime          bool          `mapstructure:"use-log-time"`
	APIEnabled          bool          `mapstructure:"api-enabled"`
	APIPort             int           `mapstructure:"api-port"`
	APIAddr             string        `mapstructure:"api-addr"`
	QueryTimeout        time.Duration `mapstructure:"query-timeout"`
	InsertBatchSize     int           `mapstructure:"insert-batch-size"`
	InsertFlushInterval time.Duration `mapstructure:"insert-flush-interval"`
	SocketPath          string        `mapstructure:"socket-path"`
	LogRetention        int           `mapstructure:"log-retention"`
	ConfigPath          string        `mapstructure:"-"` // not from config file
}
