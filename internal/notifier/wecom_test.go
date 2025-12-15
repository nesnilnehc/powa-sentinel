package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/powa-team/powa-sentinel/internal/config"
	"github.com/powa-team/powa-sentinel/internal/model"
)

func TestWeComNotifier_Send(t *testing.T) {
	// 1. Create a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify body
		var msg wecomMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		if msg.MsgType != "markdown" {
			t.Errorf("expected msgtype markdown, got %s", msg.MsgType)
		}
		if msg.Markdown == nil || msg.Markdown.Content == "" {
			t.Error("expected markdown content")
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer ts.Close()

	// 2. Configure notifier to use mock server
	cfg := &config.NotifierConfig{
		Type:       "wecom",
		WebhookURL: ts.URL,
		Retries:    1,
		RetryDelay: "10ms",
	}

	notifier, err := NewWeComNotifier(cfg)
	if err != nil {
		t.Fatalf("failed to create notifier: %v", err)
	}

	// 3. Create dummy alert
	alert := &model.AlertContext{
		ReqID: "test-req-id",
		AnalysisWindow: model.TimeWindow{
			Start: time.Now().Add(-1 * time.Hour),
			End:   time.Now(),
		},
		Summary: model.AlertSummary{
			HealthScore:          95,
			HealthStatus:         "healthy",
			TotalQueriesAnalyzed: 100,
		},
	}

	// 4. Test Send
	if err := notifier.Send(context.Background(), alert); err != nil {
		t.Errorf("Send failed: %v", err)
	}
}

func TestWeComNotifier_Retry(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer ts.Close()

	cfg := &config.NotifierConfig{
		WebhookURL: ts.URL,
		Retries:    3,
		RetryDelay: "1ms",
	}

	notifier, _ := NewWeComNotifier(cfg)
	
	if err := notifier.Send(context.Background(), &model.AlertContext{
		Summary: model.AlertSummary{HealthScore: 100},
	}); err != nil {
		t.Errorf("Send failed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestWeComNotifier_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cfg := &config.NotifierConfig{
		WebhookURL: ts.URL,
		Retries:    1,
		RetryDelay: "1ms",
	}

	notifier, _ := NewWeComNotifier(cfg)
	
	if err := notifier.Send(context.Background(), &model.AlertContext{
		Summary: model.AlertSummary{HealthScore: 100},
	}); err == nil {
		t.Error("expected error, got nil")
	}
}
