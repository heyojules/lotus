package logsource

import "github.com/tinytelemetry/lotus/internal/tcpserver"
import "github.com/tinytelemetry/lotus/internal/model"

// TCPSource wraps a tcpserver.Server as a LogSource.
type TCPSource struct {
	server *tcpserver.Server
}

// NewTCPSource creates a TCPSource from an already-started TCP server.
func NewTCPSource(server *tcpserver.Server) *TCPSource {
	return &TCPSource{server: server}
}

func (t *TCPSource) Lines() <-chan model.IngestEnvelope { return t.server.Lines() }
func (t *TCPSource) Stop()                              { _ = t.server.Stop() }
func (t *TCPSource) Name() string                       { return "tcp" }
