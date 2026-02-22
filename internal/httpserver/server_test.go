package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/control-theory/lotus/internal/duckdb"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestServer(t *testing.T) (*Server, *duckdb.Store, *gin.Engine) {
	t.Helper()
	store, err := duckdb.NewStore("")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	srv := NewServer("", store)
	srv.startTime = time.Now()

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/api/health", srv.handleHealth)
	r.GET("/api/schema", srv.handleSchema)
	r.POST("/api/query", srv.handleQuery)

	return srv, store, r
}

func TestHealthEndpoint(t *testing.T) {
	_, _, r := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal health: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("health status = %v, want ok", body["status"])
	}
}

func TestHealthEndpoint_WrongMethod(t *testing.T) {
	_, _, r := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Gin returns 405 for method not allowed when a route exists but not for this method
	if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		t.Errorf("health POST status = %d, want 405 or 404", w.Code)
	}
}

func TestQueryEndpoint_ValidSelect(t *testing.T) {
	_, store, r := newTestServer(t)

	err := store.InsertLogBatch([]*duckdb.LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "test"},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	body := `{"sql": "SELECT COUNT(*) as cnt FROM logs"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("query status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestQueryEndpoint_ValidWith(t *testing.T) {
	_, store, r := newTestServer(t)

	err := store.InsertLogBatch([]*duckdb.LogRecord{
		{Timestamp: time.Now(), Level: "INFO", Message: "test"},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	body := `{"sql": "WITH c AS (SELECT COUNT(*) as cnt FROM logs) SELECT cnt FROM c"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("query WITH status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestSchemaEndpoint(t *testing.T) {
	_, _, r := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/schema", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("schema status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestQueryEndpoint_RejectsInsert(t *testing.T) {
	_, _, r := newTestServer(t)

	body := `{"sql": "INSERT INTO logs (level, message) VALUES ('INFO', 'hack')"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("INSERT query status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestQueryEndpoint_RejectsDrop(t *testing.T) {
	_, _, r := newTestServer(t)

	body := `{"sql": "DROP TABLE logs"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("DROP query status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestQueryEndpoint_RejectsCopy(t *testing.T) {
	_, _, r := newTestServer(t)

	body := `{"sql": "SELECT 1; COPY logs TO '/tmp/evil.csv'"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("COPY query status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestQueryEndpoint_RejectsAttach(t *testing.T) {
	_, _, r := newTestServer(t)

	body := `{"sql": "SELECT 1; ATTACH '/tmp/evil.db'"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ATTACH query status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestQueryEndpoint_WrongMethod(t *testing.T) {
	_, _, r := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Gin returns 405 for method not allowed
	if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		t.Errorf("query GET status = %d, want 405 or 404", w.Code)
	}
}

func TestQueryEndpoint_EmptySQL(t *testing.T) {
	_, _, r := newTestServer(t)

	body := `{"sql": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("empty sql status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGinRecovery(t *testing.T) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic recovery status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
