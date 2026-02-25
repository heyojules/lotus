package tcpserver

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/control-theory/lotus/internal/model"
)

const (
	// DefaultLineChannelSize is the default buffer size for the incoming log line channel.
	DefaultLineChannelSize = 100_000

	// DefaultMaxLineSize is the default maximum size (in bytes) of a single log line.
	DefaultMaxLineSize = 1024 * 1024 // 1MB
)

// ServerConfig holds tunable parameters for the TCP server.
type ServerConfig struct {
	LineChannelSize int
	MaxLineSize     int
}

// Server listens for newline-delimited OTEL JSON log payloads over TCP.
type Server struct {
	listener    net.Listener
	addr        string
	lineChan    chan model.IngestEnvelope
	maxLineSize int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewServer creates a new TCP server. Default addr is "127.0.0.1:4000".
func NewServer(addr string, conf ...ServerConfig) *Server {
	if addr == "" {
		addr = "127.0.0.1:4000"
	}
	lineChannelSize := DefaultLineChannelSize
	maxLineSize := DefaultMaxLineSize
	if len(conf) > 0 {
		if conf[0].LineChannelSize > 0 {
			lineChannelSize = conf[0].LineChannelSize
		}
		if conf[0].MaxLineSize > 0 {
			maxLineSize = conf[0].MaxLineSize
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		addr:        addr,
		lineChan:    make(chan model.IngestEnvelope, lineChannelSize),
		maxLineSize: maxLineSize,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins accepting TCP connections.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = listener

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-s.ctx.Done():
					return
				default:
					continue
				}
			}
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}()

	return nil
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	buf := make([]byte, s.maxLineSize)
	scanner.Buffer(buf, s.maxLineSize)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		select {
		case s.lineChan <- model.IngestEnvelope{Source: "tcp", Line: line}:
		case <-s.ctx.Done():
			return
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			log.Printf("tcpserver: dropped connection %s due to line exceeding max size (%d bytes)", conn.RemoteAddr(), s.maxLineSize)
			return
		}
		log.Printf("tcpserver: scanner error from %s: %v", conn.RemoteAddr(), err)
	}
}

// Stop gracefully shuts down the TCP server.
func (s *Server) Stop() error {
	s.cancel()
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	close(s.lineChan)
	return nil
}

// Lines returns the channel of received log lines.
func (s *Server) Lines() <-chan model.IngestEnvelope {
	return s.lineChan
}

// Addr returns the active listen address.
// Before Start, it returns the configured address.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}
