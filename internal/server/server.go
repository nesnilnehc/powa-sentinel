// Package server provides the HTTP server for health checks and metrics.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/powa-team/powa-sentinel/internal/config"
	"github.com/powa-team/powa-sentinel/internal/reader"
)

// Server provides HTTP endpoints for health checks and monitoring.
type Server struct {
	cfg      *config.ServerConfig
	reader   *reader.Reader
	server   *http.Server
	mu       sync.Mutex
	started  time.Time
	healthy  bool
	lastPing time.Time
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Uptime    string    `json:"uptime,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Database  *DBHealth `json:"database,omitempty"`
}

// DBHealth represents database connectivity status.
type DBHealth struct {
	Connected bool   `json:"connected"`
	Latency   string `json:"latency,omitempty"`
	Error     string `json:"error,omitempty"`
}

// New creates a new Server.
func New(cfg *config.ServerConfig, r *reader.Reader) *Server {
	return &Server{
		cfg:     cfg,
		reader:  r,
		healthy: true,
	}
}

// Start begins serving HTTP requests.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)
	mux.HandleFunc("/livez", s.handleLive)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	s.started = time.Now()

	go func() {
		log.Printf("Health server listening on :%d", s.cfg.Port)
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	return s.server.Shutdown(ctx)
}

// handleHealth handles /healthz endpoint (combined check).
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Uptime:    time.Since(s.started).Round(time.Second).String(),
	}

	// Perform deep check if enabled
	if s.cfg.DeepCheck && s.reader != nil {
		dbHealth := s.checkDatabase(r.Context())
		response.Database = dbHealth
		if !dbHealth.Connected {
			response.Status = "degraded"
		}
	}

	statusCode := http.StatusOK
	if response.Status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}

	s.writeJSON(w, statusCode, response)
}

// handleReady handles /readyz endpoint (readiness probe).
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if we can connect to the database
	if s.reader != nil {
		dbHealth := s.checkDatabase(r.Context())
		if !dbHealth.Connected {
			s.writeJSON(w, http.StatusServiceUnavailable, HealthResponse{
				Status:    "not ready",
				Timestamp: time.Now(),
				Database:  dbHealth,
			})
			return
		}
	}

	s.writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "ready",
		Timestamp: time.Now(),
	})
}

// handleLive handles /livez endpoint (liveness probe).
func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - if we can respond, we're alive
	s.writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "alive",
		Timestamp: time.Now(),
		Uptime:    time.Since(s.started).Round(time.Second).String(),
	})
}

// checkDatabase tests database connectivity.
func (s *Server) checkDatabase(ctx context.Context) *DBHealth {
	health := &DBHealth{}

	start := time.Now()
	err := s.reader.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		health.Connected = false
		health.Error = err.Error()
	} else {
		health.Connected = true
		health.Latency = latency.String()
		s.lastPing = time.Now()
	}

	return health
}

// writeJSON writes a JSON response.
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}
