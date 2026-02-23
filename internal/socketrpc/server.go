package socketrpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/control-theory/lotus/internal/model"
)

const (
	// scannerInitBufSize is the initial buffer size for the per-connection scanner (1 MB).
	scannerInitBufSize = 1024 * 1024
	// scannerMaxTokenSize is the maximum token size the scanner will accept (10 MB).
	scannerMaxTokenSize = 10 * 1024 * 1024
)

// Server exposes a model.LogQuerier over a Unix domain socket using JSON-RPC 2.0.
type Server struct {
	socketPath string
	store      model.LogQuerier
	listener   net.Listener
	wg         sync.WaitGroup
	quit       chan struct{}
}

// NewServer creates a new socket RPC server.
func NewServer(socketPath string, store model.LogQuerier) *Server {
	return &Server{
		socketPath: socketPath,
		store:      store,
		quit:       make(chan struct{}),
	}
}

// Start begins listening on the Unix socket and accepting connections.
func (s *Server) Start() error {
	// Ensure the parent directory exists.
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0755); err != nil {
		return fmt.Errorf("socketrpc: mkdir: %w", err)
	}

	// Remove stale socket if it exists.
	if _, err := os.Stat(s.socketPath); err == nil {
		conn, dialErr := net.DialTimeout("unix", s.socketPath, 500*time.Millisecond)
		if dialErr != nil {
			// Socket file exists but nobody is listening — stale.
			os.Remove(s.socketPath)
		} else {
			conn.Close()
			return fmt.Errorf("socketrpc: another server is already listening on %s", s.socketPath)
		}
	}

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("socketrpc: listen: %w", err)
	}
	s.listener = ln

	s.wg.Add(1)
	go s.acceptLoop()

	log.Printf("socketrpc: listening on %s", s.socketPath)
	return nil
}

// Stop closes the listener, waits for connections to drain, and removes the socket file.
func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	os.Remove(s.socketPath)
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("socketrpc: accept error: %v", err)
				// Continue on transient errors (e.g., fd limit) instead of
				// killing the entire accept loop.
				continue
			}
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, scannerInitBufSize), scannerMaxTokenSize)
	encoder := json.NewEncoder(conn)

	for scanner.Scan() {
		select {
		case <-s.quit:
			return
		default:
		}

		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			resp := Response{JSONRPC: "2.0", ID: 0, Error: &RPCError{Code: -32700, Message: "parse error"}}
			encoder.Encode(resp)
			continue
		}

		resp := s.dispatch(req)
		if err := encoder.Encode(resp); err != nil {
			return
		}
	}
}

func (s *Server) dispatch(req Request) Response {
	resp := Response{JSONRPC: "2.0", ID: req.ID}

	marshalResult := func(v interface{}, err error) Response {
		if err != nil {
			resp.Error = &RPCError{Code: -32000, Message: err.Error()}
			return resp
		}
		data, merr := json.Marshal(v)
		if merr != nil {
			resp.Error = &RPCError{Code: -32603, Message: merr.Error()}
			return resp
		}
		resp.Result = data
		return resp
	}

	invalidParams := func(err error) Response {
		resp.Error = &RPCError{Code: -32602, Message: fmt.Sprintf("invalid params: %v", err)}
		return resp
	}

	switch req.Method {
	case "TotalLogCount":
		var p struct{ Opts model.QueryOpts }
		if err := json.Unmarshal(req.Params, &p); err != nil && len(req.Params) > 0 {
			return invalidParams(err)
		}
		return marshalResult(s.store.TotalLogCount(p.Opts))

	case "TotalLogBytes":
		var p struct{ Opts model.QueryOpts }
		if err := json.Unmarshal(req.Params, &p); err != nil && len(req.Params) > 0 {
			return invalidParams(err)
		}
		return marshalResult(s.store.TotalLogBytes(p.Opts))

	case "TopWords":
		var p struct {
			Limit int
			Opts  model.QueryOpts
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return invalidParams(err)
		}
		return marshalResult(s.store.TopWords(p.Limit, p.Opts))

	case "TopAttributes":
		var p struct {
			Limit int
			Opts  model.QueryOpts
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return invalidParams(err)
		}
		return marshalResult(s.store.TopAttributes(p.Limit, p.Opts))

	case "TopAttributeKeys":
		var p struct {
			Limit int
			Opts  model.QueryOpts
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return invalidParams(err)
		}
		return marshalResult(s.store.TopAttributeKeys(p.Limit, p.Opts))

	case "AttributeKeyValues":
		var p struct {
			Key   string
			Limit int
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return invalidParams(err)
		}
		return marshalResult(s.store.AttributeKeyValues(p.Key, p.Limit))

	case "SeverityCounts":
		var p struct{ Opts model.QueryOpts }
		if err := json.Unmarshal(req.Params, &p); err != nil && len(req.Params) > 0 {
			return invalidParams(err)
		}
		return marshalResult(s.store.SeverityCounts(p.Opts))

	case "SeverityCountsByMinute":
		var p struct {
			Window time.Duration
			Opts   model.QueryOpts
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return invalidParams(err)
		}
		return marshalResult(s.store.SeverityCountsByMinute(p.Window, p.Opts))

	case "TopHosts":
		var p struct {
			Limit int
			Opts  model.QueryOpts
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return invalidParams(err)
		}
		return marshalResult(s.store.TopHosts(p.Limit, p.Opts))

	case "TopServices":
		var p struct {
			Limit int
			Opts  model.QueryOpts
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return invalidParams(err)
		}
		return marshalResult(s.store.TopServices(p.Limit, p.Opts))

	case "TopServicesBySeverity":
		var p struct {
			Severity string
			Limit    int
			Opts     model.QueryOpts
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return invalidParams(err)
		}
		return marshalResult(s.store.TopServicesBySeverity(p.Severity, p.Limit, p.Opts))

	case "ListApps":
		return marshalResult(s.store.ListApps())

	case "RecentLogsFiltered":
		var p struct {
			Limit          int
			App            string
			SeverityLevels []string
			MessagePattern string
		}
		// Allow empty/null params for defaults; only reject genuinely malformed JSON.
		if err := json.Unmarshal(req.Params, &p); err != nil && len(req.Params) > 0 {
			return invalidParams(err)
		}
		return marshalResult(s.store.RecentLogsFiltered(p.Limit, p.App, p.SeverityLevels, p.MessagePattern))

	default:
		resp.Error = &RPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
		return resp
	}
}
