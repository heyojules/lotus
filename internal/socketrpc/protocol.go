package socketrpc

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// JSON-RPC 2.0 Method Reference
//
// The socket RPC server exposes model.LogQuerier over Unix domain socket.
// Each method maps 1:1 to the LogQuerier interface.
//
//   Method                    Params                                              Result
//   ──────────────────────    ──────────────────────────────────────────────────   ─────────────────────────
//   TotalLogCount             {Opts: QueryOpts}                                   int64
//   TotalLogBytes             {Opts: QueryOpts}                                   int64
//   TopWords                  {Limit: int, Opts: QueryOpts}                       []WordCount
//   TopAttributes             {Limit: int, Opts: QueryOpts}                       []AttributeStat
//   TopAttributeKeys          {Limit: int, Opts: QueryOpts}                       []AttributeKeyStat
//   AttributeKeyValues        {Key: string, Limit: int}                           map[string]int64
//   SeverityCounts            {Opts: QueryOpts}                                   map[string]int64
//   SeverityCountsByMinute    {Window: time.Duration, Opts: QueryOpts}            []MinuteCounts
//   TopHosts                  {Limit: int, Opts: QueryOpts}                       []DimensionCount
//   TopServices               {Limit: int, Opts: QueryOpts}                       []DimensionCount
//   TopServicesBySeverity     {Severity: string, Limit: int, Opts: QueryOpts}     []DimensionCount
//   ListApps                  (none)                                              []string
//   RecentLogsFiltered        {Limit: int, App: string, SeverityLevels: []string, MessagePattern: string}  []LogRecord
//
// QueryOpts: {App: string} — empty string means all apps.
// Methods with optional params (TotalLogCount, TotalLogBytes, SeverityCounts,
// RecentLogsFiltered) accept empty or null params gracefully.
//
// Error codes follow JSON-RPC 2.0:
//   -32700  Parse error (malformed JSON)
//   -32601  Method not found
//   -32602  Invalid params
//   -32603  Internal error (marshal failure)
//   -32000  Application error (query failure)

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return e.Message }

// DefaultSocketPath returns the default Unix socket path.
// It prefers $XDG_RUNTIME_DIR/lotus/lotus.sock, falling back to
// ~/.config/lotus/lotus.sock.
func DefaultSocketPath() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "lotus", "lotus.sock")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/lotus.sock"
	}
	return filepath.Join(home, ".local", "state", "lotus", "lotus.sock")
}
