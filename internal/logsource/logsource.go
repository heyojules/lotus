package logsource

import "github.com/control-theory/lotus/internal/model"

// LogSource is a unified interface for all log input sources (TCP, file, stdin).
type LogSource interface {
	Lines() <-chan model.IngestEnvelope // read-only channel of log lines
	Stop()                              // graceful shutdown
	Name() string                       // "tcp", "file", "stdin"
}
