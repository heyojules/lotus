package logsource

// LogSource is a unified interface for all log input sources (TCP, file, stdin).
type LogSource interface {
	Lines() <-chan string // read-only channel of log lines
	Stop()               // graceful shutdown
	Name() string        // "tcp", "file", "stdin"
}
