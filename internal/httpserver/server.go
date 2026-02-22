package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/control-theory/lotus/internal/model"
	"github.com/gin-gonic/gin"
)

// QueryStore is the narrow store contract required by the HTTP API.
type QueryStore interface {
	model.SchemaQuerier
	TotalLogCount(opts model.QueryOpts) (int64, error)
}

// Server provides an HTTP API for querying Lotus analytics.
type Server struct {
	addr      string
	store     QueryStore
	server    *http.Server
	ctx       context.Context
	cancel    context.CancelFunc
	startTime time.Time
}

// NewServer creates a new HTTP API server.
func NewServer(addr string, store QueryStore) *Server {
	if addr == "" {
		addr = "0.0.0.0:3000"
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		addr:   addr,
		store:  store,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins serving HTTP requests.
func (s *Server) Start() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/api/health", s.handleHealth)
	r.GET("/api/schema", s.handleSchema)
	r.POST("/api/query", s.handleQuery)

	s.server = &http.Server{
		Handler:           r,
		BaseContext:       func(_ net.Listener) context.Context { return s.ctx },
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.startTime = time.Now()

	go s.server.Serve(listener)
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop() error {
	s.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *Server) handleHealth(c *gin.Context) {
	logCount, err := s.store.TotalLogCount(model.QueryOpts{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read health metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"uptime":    time.Since(s.startTime).String(),
		"log_count": logCount,
	})
}

func (s *Server) handleSchema(c *gin.Context) {
	description := s.store.GetSchemaDescription()

	tables, err := s.store.ExecuteQuery(
		"SELECT table_name, column_name, data_type FROM information_schema.columns WHERE table_schema = 'main' ORDER BY table_name, ordinal_position",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read schema metadata"})
		return
	}

	schema := make(map[string][]map[string]string)
	for _, row := range tables {
		tableName := fmt.Sprintf("%v", row["table_name"])
		schema[tableName] = append(schema[tableName], map[string]string{
			"column": fmt.Sprintf("%v", row["column_name"]),
			"type":   fmt.Sprintf("%v", row["data_type"]),
		})
	}

	counts, err := s.store.TableRowCounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read table row counts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"description": description,
		"tables":      schema,
		"row_counts":  counts,
	})
}

func (s *Server) handleQuery(c *gin.Context) {
	var req struct {
		SQL string `json:"sql" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body or missing sql field"})
		return
	}

	results, err := s.store.ExecuteQuery(req.SQL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var columns []string
	if len(results) > 0 {
		for col := range results[0] {
			columns = append(columns, col)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"columns":   columns,
		"rows":      results,
		"row_count": len(results),
	})
}
