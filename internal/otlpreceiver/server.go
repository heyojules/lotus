package otlpreceiver

import (
	"log"
	"net"
	"sync"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"

	"github.com/tinytelemetry/tiny-telemetry/internal/model"
)

// Server is an OTLP/gRPC log receiver.
type Server struct {
	addr     string
	sink     model.RecordSink
	grpc     *grpc.Server
	listener net.Listener
	stopOnce sync.Once
}

// NewServer creates a new OTLP/gRPC server.
func NewServer(addr string, sink model.RecordSink) *Server {
	return &Server{
		addr: addr,
		sink: sink,
	}
}

// Start begins listening and serving gRPC in a background goroutine.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = ln

	s.grpc = grpc.NewServer(
		grpc.MaxRecvMsgSize(16 << 20), // 16 MB for large OTLP batches
	)
	collogspb.RegisterLogsServiceServer(s.grpc, &logsHandler{sink: s.sink})

	go func() {
		if err := s.grpc.Serve(ln); err != nil {
			log.Printf("otlpreceiver: grpc.Serve exited: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the gRPC server.
func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		if s.grpc != nil {
			s.grpc.GracefulStop()
		}
	})
}

// Addr returns the actual listen address (useful when port 0 is used).
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}
