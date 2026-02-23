package socketrpc

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/control-theory/lotus/internal/model"
)

// stubQuerier returns fixed values for dispatch unit testing.
type stubQuerier struct{}

func (q *stubQuerier) TotalLogCount(opts model.QueryOpts) (int64, error) { return 100, nil }
func (q *stubQuerier) TotalLogBytes(opts model.QueryOpts) (int64, error) { return 4096, nil }
func (q *stubQuerier) TopWords(limit int, opts model.QueryOpts) ([]model.WordCount, error) {
	return []model.WordCount{{Word: "error", Count: 5}}, nil
}
func (q *stubQuerier) TopAttributes(limit int, opts model.QueryOpts) ([]model.AttributeStat, error) {
	return []model.AttributeStat{{Key: "env", Value: "prod", Count: 3}}, nil
}
func (q *stubQuerier) TopAttributeKeys(limit int, opts model.QueryOpts) ([]model.AttributeKeyStat, error) {
	return []model.AttributeKeyStat{{Key: "env", UniqueValues: 2, TotalCount: 10}}, nil
}
func (q *stubQuerier) AttributeKeyValues(key string, limit int) (map[string]int64, error) {
	return map[string]int64{"prod": 7}, nil
}
func (q *stubQuerier) SeverityCounts(opts model.QueryOpts) (map[string]int64, error) {
	return map[string]int64{"INFO": 50, "ERROR": 10}, nil
}
func (q *stubQuerier) SeverityCountsByMinute(opts model.QueryOpts) ([]model.MinuteCounts, error) {
	return []model.MinuteCounts{{Minute: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Info: 5, Total: 5}}, nil
}
func (q *stubQuerier) TopHosts(limit int, opts model.QueryOpts) ([]model.DimensionCount, error) {
	return []model.DimensionCount{{Value: "host1", Count: 20}}, nil
}
func (q *stubQuerier) TopServices(limit int, opts model.QueryOpts) ([]model.DimensionCount, error) {
	return []model.DimensionCount{{Value: "api", Count: 15}}, nil
}
func (q *stubQuerier) TopServicesBySeverity(severity string, limit int, opts model.QueryOpts) ([]model.DimensionCount, error) {
	return []model.DimensionCount{{Value: "api", Count: 8}}, nil
}
func (q *stubQuerier) ListApps() ([]string, error) { return []string{"default"}, nil }
func (q *stubQuerier) RecentLogsFiltered(limit int, app string, severityLevels []string, messagePattern string) ([]model.LogRecord, error) {
	return []model.LogRecord{{
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Level:     "INFO",
		Message:   "test",
		App:       "default",
	}}, nil
}
func (q *stubQuerier) ExecuteQuery(query string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{{"ok": true}}, nil
}
func (q *stubQuerier) GetSchemaDescription() string { return "schema" }
func (q *stubQuerier) TableRowCounts() (map[string]int64, error) {
	return map[string]int64{"logs": 1}, nil
}

func newTestDispatcher() *Server {
	return &Server{store: &stubQuerier{}}
}

func TestDispatch_AllMethods(t *testing.T) {
	t.Parallel()
	srv := newTestDispatcher()

	tests := []struct {
		method string
		params string
	}{
		{"TotalLogCount", `{"Opts":{}}`},
		{"TotalLogBytes", `{"Opts":{}}`},
		{"TopWords", `{"Limit":10,"Opts":{}}`},
		{"TopAttributes", `{"Limit":10,"Opts":{}}`},
		{"TopAttributeKeys", `{"Limit":10,"Opts":{}}`},
		{"AttributeKeyValues", `{"Key":"env","Limit":10}`},
		{"SeverityCounts", `{"Opts":{}}`},
		{"SeverityCountsByMinute", `{"Opts":{}}`},
		{"TopHosts", `{"Limit":10,"Opts":{}}`},
		{"TopServices", `{"Limit":10,"Opts":{}}`},
		{"TopServicesBySeverity", `{"Severity":"ERROR","Limit":10,"Opts":{}}`},
		{"ListApps", `{}`},
		{"RecentLogsFiltered", `{"Limit":100}`},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()
			req := Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  tt.method,
				Params:  json.RawMessage(tt.params),
			}
			resp := srv.dispatch(req)
			if resp.Error != nil {
				t.Fatalf("dispatch(%s) error: %s", tt.method, resp.Error.Message)
			}
			if resp.Result == nil {
				t.Fatalf("dispatch(%s) returned nil result", tt.method)
			}
			if resp.JSONRPC != "2.0" {
				t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
			}
			if resp.ID != 1 {
				t.Errorf("ID = %d, want 1", resp.ID)
			}
		})
	}
}

func TestDispatch_MethodNotFound(t *testing.T) {
	t.Parallel()
	srv := newTestDispatcher()

	resp := srv.dispatch(Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "NonExistentMethod",
		Params:  json.RawMessage(`{}`),
	})
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
}

func TestDispatch_InvalidParams(t *testing.T) {
	t.Parallel()
	srv := newTestDispatcher()

	// TopWords requires Limit param — send garbage JSON
	resp := srv.dispatch(Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "TopWords",
		Params:  json.RawMessage(`not json`),
	})
	if resp.Error == nil {
		t.Fatal("expected error for malformed params")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want -32602 (invalid params)", resp.Error.Code)
	}
}

func TestDispatch_EmptyParamsOnOptionalMethods(t *testing.T) {
	t.Parallel()
	srv := newTestDispatcher()

	// TotalLogCount, TotalLogBytes, SeverityCounts, and RecentLogsFiltered
	// accept empty/null params gracefully.
	methods := []string{"TotalLogCount", "TotalLogBytes", "SeverityCounts", "RecentLogsFiltered"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			resp := srv.dispatch(Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  method,
				Params:  nil,
			})
			if resp.Error != nil {
				t.Fatalf("dispatch(%s) with nil params: %s", method, resp.Error.Message)
			}
		})
	}
}

func TestDispatch_PreservesRequestID(t *testing.T) {
	t.Parallel()
	srv := newTestDispatcher()

	for _, id := range []int{0, 1, 42, 9999} {
		resp := srv.dispatch(Request{
			JSONRPC: "2.0",
			ID:      id,
			Method:  "ListApps",
			Params:  json.RawMessage(`{}`),
		})
		if resp.ID != id {
			t.Errorf("request ID %d: response ID = %d", id, resp.ID)
		}
	}
}
