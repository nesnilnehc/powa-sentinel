package engine

import (
	"testing"

	"github.com/powa-team/powa-sentinel/internal/config"
	"github.com/powa-team/powa-sentinel/internal/model"
)

func TestAnalyzeSlowSQL(t *testing.T) {
	cfg := &config.Config{
		Rules: config.RulesConfig{
			SlowSQL: config.SlowSQLRuleConfig{
				TopN:   3,
				RankBy: "total_time",
			},
		},
	}
	eng := New(cfg, nil)

	metrics := []model.MetricSnapshot{
		{QueryID: 1, TotalTime: 100, MeanTime: 10},
		{QueryID: 2, TotalTime: 500, MeanTime: 50},
		{QueryID: 3, TotalTime: 200, MeanTime: 20},
		{QueryID: 4, TotalTime: 300, MeanTime: 30},
		{QueryID: 5, TotalTime: 400, MeanTime: 40},
	}

	result := eng.analyzeSlowSQL(metrics)

	if len(result) != 3 {
		t.Errorf("analyzeSlowSQL() returned %d items, want 3", len(result))
	}

	// Check ordering (descending by total_time)
	if result[0].QueryID != 2 {
		t.Errorf("First slow query should be QueryID 2, got %d", result[0].QueryID)
	}
	if result[1].QueryID != 5 {
		t.Errorf("Second slow query should be QueryID 5, got %d", result[1].QueryID)
	}
	if result[2].QueryID != 4 {
		t.Errorf("Third slow query should be QueryID 4, got %d", result[2].QueryID)
	}
}

func TestAnalyzeSlowSQL_RankByMeanTime(t *testing.T) {
	cfg := &config.Config{
		Rules: config.RulesConfig{
			SlowSQL: config.SlowSQLRuleConfig{
				TopN:   2,
				RankBy: "mean_time",
			},
		},
	}
	eng := New(cfg, nil)

	metrics := []model.MetricSnapshot{
		{QueryID: 1, TotalTime: 1000, MeanTime: 10},
		{QueryID: 2, TotalTime: 100, MeanTime: 100},
		{QueryID: 3, TotalTime: 500, MeanTime: 50},
	}

	result := eng.analyzeSlowSQL(metrics)

	if len(result) != 2 {
		t.Errorf("analyzeSlowSQL() returned %d items, want 2", len(result))
	}

	// Check ordering (descending by mean_time)
	if result[0].QueryID != 2 {
		t.Errorf("First slow query should be QueryID 2 (highest mean_time), got %d", result[0].QueryID)
	}
	if result[1].QueryID != 3 {
		t.Errorf("Second slow query should be QueryID 3, got %d", result[1].QueryID)
	}
}

func TestDetectRegressions(t *testing.T) {
	cfg := &config.Config{
		Rules: config.RulesConfig{
			Regression: config.RegressionRuleConfig{
				ThresholdPercent: 50,
			},
		},
	}
	eng := New(cfg, nil)

	current := []model.MetricSnapshot{
		{QueryID: 1, MeanTime: 150}, // 50% increase
		{QueryID: 2, MeanTime: 250}, // 150% increase
		{QueryID: 3, MeanTime: 100}, // No change
		{QueryID: 4, MeanTime: 80},  // 20% decrease
	}

	baseline := []model.MetricSnapshot{
		{QueryID: 1, MeanTime: 100},
		{QueryID: 2, MeanTime: 100},
		{QueryID: 3, MeanTime: 100},
		{QueryID: 4, MeanTime: 100},
	}

	result := eng.detectRegressions(current, baseline)

	if len(result) != 2 {
		t.Errorf("detectRegressions() returned %d items, want 2", len(result))
	}

	// Should be sorted by change percent descending
	if result[0].QueryID != 2 {
		t.Errorf("First regression should be QueryID 2 (150%% increase), got %d", result[0].QueryID)
	}
	if result[1].QueryID != 1 {
		t.Errorf("Second regression should be QueryID 1 (50%% increase), got %d", result[1].QueryID)
	}
}

func TestDetectRegressions_EdgeCases(t *testing.T) {
	cfg := &config.Config{
		Rules: config.RulesConfig{
			Regression: config.RegressionRuleConfig{
				ThresholdPercent: 50,
			},
		},
	}
	eng := New(cfg, nil)

	t.Run("empty current", func(t *testing.T) {
		result := eng.detectRegressions(nil, []model.MetricSnapshot{{QueryID: 1, MeanTime: 100}})
		if result != nil {
			t.Errorf("Expected nil for empty current, got %v", result)
		}
	})

	t.Run("empty baseline", func(t *testing.T) {
		result := eng.detectRegressions([]model.MetricSnapshot{{QueryID: 1, MeanTime: 100}}, nil)
		if result != nil {
			t.Errorf("Expected nil for empty baseline, got %v", result)
		}
	})

	t.Run("zero baseline mean time", func(t *testing.T) {
		current := []model.MetricSnapshot{{QueryID: 1, MeanTime: 100}}
		baseline := []model.MetricSnapshot{{QueryID: 1, MeanTime: 0}}
		result := eng.detectRegressions(current, baseline)
		if len(result) != 0 {
			t.Errorf("Expected 0 regressions for zero baseline, got %d", len(result))
		}
	})
}

func TestFilterSuggestions(t *testing.T) {
	cfg := &config.Config{
		Rules: config.RulesConfig{
			IndexSuggestion: config.IndexSuggestionRuleConfig{
				MinImprovementPercent: 30,
			},
		},
	}
	eng := New(cfg, nil)

	suggestions := []model.IndexSuggestion{
		{Table: "users", EstImprovementPercent: 50},
		{Table: "orders", EstImprovementPercent: 20},
		{Table: "products", EstImprovementPercent: 35},
		{Table: "logs", EstImprovementPercent: 30},
	}

	result := eng.filterSuggestions(suggestions)

	if len(result) != 3 {
		t.Errorf("filterSuggestions() returned %d items, want 3", len(result))
	}

	// Should be sorted by improvement descending
	if result[0].Table != "users" {
		t.Errorf("First suggestion should be 'users', got %s", result[0].Table)
	}
}

func TestGenerateSummary(t *testing.T) {
	cfg := &config.Config{}
	eng := New(cfg, nil)

	t.Run("healthy score", func(t *testing.T) {
		alertCtx := &model.AlertContext{
			TopSlowSQL:  make([]model.MetricSnapshot, 5),
			Regressions: nil,
			Suggestions: nil,
		}
		summary := eng.generateSummary(alertCtx, 100)

		if summary.HealthScore != 100 {
			t.Errorf("HealthScore = %d, want 100", summary.HealthScore)
		}
		if summary.HealthStatus != "healthy" {
			t.Errorf("HealthStatus = %s, want healthy", summary.HealthStatus)
		}
	})

	t.Run("deductions with caps", func(t *testing.T) {
		alertCtx := &model.AlertContext{
			TopSlowSQL: make([]model.MetricSnapshot, 5),
			Regressions: []model.RegressionItem{
				{Severity: "critical"},
				{Severity: "critical"},
				{Severity: "critical"},
				{Severity: "critical"}, // 80 points total, should cap at 50
			},
			Suggestions: make([]model.IndexSuggestion, 20), // 60 points, should cap at 30
		}
		summary := eng.generateSummary(alertCtx, 100)

		// Score should be 100 - 50 (capped regressions) - 30 (capped suggestions) = 20
		if summary.HealthScore != 20 {
			t.Errorf("HealthScore = %d, want 20 (with caps applied)", summary.HealthScore)
		}
		if summary.HealthStatus != "critical" {
			t.Errorf("HealthStatus = %s, want critical", summary.HealthStatus)
		}
	})

	t.Run("score floor at 0", func(t *testing.T) {
		alertCtx := &model.AlertContext{
			Regressions: []model.RegressionItem{
				{Severity: "critical"},
				{Severity: "critical"},
				{Severity: "critical"},
			},
			Suggestions: make([]model.IndexSuggestion, 20),
		}
		summary := eng.generateSummary(alertCtx, 100)

		if summary.HealthScore < 0 {
			t.Errorf("HealthScore = %d, should not be negative", summary.HealthScore)
		}
	})
}

func TestCalculateSeverity(t *testing.T) {
	tests := []struct {
		changePercent float64
		expected      string
	}{
		{600, "critical"},
		{500, "critical"},
		{300, "high"},
		{200, "high"},
		{150, "medium"},
		{100, "medium"},
		{75, "low"},
		{50, "low"},
		{0, "low"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := calculateSeverity(tt.changePercent)
			if result != tt.expected {
				t.Errorf("calculateSeverity(%f) = %s, want %s", tt.changePercent, result, tt.expected)
			}
		})
	}
}

func TestGetHealthStatus(t *testing.T) {
	tests := []struct {
		score    int
		expected string
	}{
		{100, "healthy"},
		{90, "healthy"},
		{89, "warning"},
		{70, "warning"},
		{69, "degraded"},
		{50, "degraded"},
		{49, "critical"},
		{0, "critical"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getHealthStatus(tt.score)
			if result != tt.expected {
				t.Errorf("getHealthStatus(%d) = %s, want %s", tt.score, result, tt.expected)
			}
		})
	}
}
