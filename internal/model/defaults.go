package model

import "time"

// Shared defaults used by both the server and CLI binaries.
const (
	DefaultUpdateInterval = 2 * time.Second
	DefaultLogBuffer      = 1000
	DefaultSkin           = "default"
)
