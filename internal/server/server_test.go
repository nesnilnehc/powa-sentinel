package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/powa-team/powa-sentinel/internal/config"
)

func TestHealthEndpoints(t *testing.T) {
	cfg := &config.ServerConfig{
		Port:      8080,
		DeepCheck: false, // Disable deep check for tests without DB
	}

	srv := New(cfg, nil)

	t.Run("GET /livez", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/livez", nil)
		w := httptest.NewRecorder()

		srv.handleLive(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var health HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if health.Status != "alive" {
			t.Errorf("Status = %s, want alive", health.Status)
		}

		if health.Uptime == "" {
			t.Error("Uptime should not be empty")
		}
	})

	t.Run("GET /healthz without deep check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()

		srv.handleHealth(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var health HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if health.Status != "ok" {
			t.Errorf("Status = %s, want ok", health.Status)
		}

		// Database should not be checked when deep check is disabled
		if health.Database != nil {
			t.Error("Database should be nil when deep check is disabled")
		}
	})

	t.Run("GET /readyz without DB", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/readyz", nil)
		w := httptest.NewRecorder()

		srv.handleReady(w, req)

		resp := w.Result()
		// Should be OK when no reader is configured (no DB to check)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var health HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if health.Status != "ready" {
			t.Errorf("Status = %s, want ready", health.Status)
		}
	})
}

func TestHealthResponse_JSON(t *testing.T) {
	cfg := &config.ServerConfig{
		Port:      8080,
		DeepCheck: false,
	}

	srv := New(cfg, nil)

	req := httptest.NewRequest("GET", "/livez", nil)
	w := httptest.NewRecorder()

	srv.handleLive(w, req)

	resp := w.Result()

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", contentType)
	}

	// Verify it's valid JSON
	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	// Timestamp should be set
	if health.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}
