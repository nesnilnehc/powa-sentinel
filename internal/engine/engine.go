// Package engine contains the analysis logic for detecting performance issues.
package engine

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/powa-team/powa-sentinel/internal/config"
	"github.com/powa-team/powa-sentinel/internal/model"
	"github.com/powa-team/powa-sentinel/internal/reader"
)

// Scoring limits to prevent single dimension from dominating
const (
	MaxRegressionDeduction  = 50 // Maximum points deducted for regressions
	MaxSuggestionDeduction  = 30 // Maximum points deducted for index suggestions
)

// Engine performs analysis on PoWA data and generates alerts.
type Engine struct {
	cfg    *config.Config
	reader *reader.Reader
}

// New creates a new Engine with the given configuration and reader.
func New(cfg *config.Config, r *reader.Reader) *Engine {
	return &Engine{
		cfg:    cfg,
		reader: r,
	}
}

// Analyze runs the complete analysis and returns an AlertContext.
func (e *Engine) Analyze(ctx context.Context) (*model.AlertContext, error) {
	// Parse time windows
	windowDuration, err := e.cfg.Analysis.WindowDurationParsed()
	if err != nil {
		return nil, fmt.Errorf("parsing window duration: %w", err)
	}

	comparisonOffset, err := e.cfg.Analysis.ComparisonOffsetParsed()
	if err != nil {
		return nil, fmt.Errorf("parsing comparison offset: %w", err)
	}

	// Fetch current metrics
	currentMetrics, err := e.reader.GetCurrentMetrics(ctx, windowDuration)
	if err != nil {
		return nil, fmt.Errorf("fetching current metrics: %w", err)
	}

	// Fetch baseline metrics
	baselineMetrics, err := e.reader.GetBaselineMetrics(ctx, comparisonOffset, windowDuration)
	if err != nil {
		return nil, fmt.Errorf("fetching baseline metrics: %w", err)
	}

	// Fetch index suggestions (non-fatal error)
	suggestions, err := e.reader.GetIndexSuggestions(ctx)
	if err != nil {
		// Log the error but continue without suggestions
		log.Printf("Warning: failed to fetch index suggestions: %v", err)
		suggestions = nil
	}

	// Build time windows
	now := time.Now()
	analysisWindow := model.TimeWindow{
		Start: now.Add(-windowDuration),
		End:   now,
	}
	baselineWindow := model.TimeWindow{
		Start: now.Add(-comparisonOffset - windowDuration),
		End:   now.Add(-comparisonOffset),
	}

	// Create alert context
	alertCtx := &model.AlertContext{
		ReqID:          generateReqID(),
		ReportType:     "scheduled",
		Timestamp:      now,
		AnalysisWindow: analysisWindow,
		BaselineWindow: baselineWindow,
	}

	// Run analysis rules
	alertCtx.TopSlowSQL = e.analyzeSlowSQL(currentMetrics)
	alertCtx.Regressions = e.detectRegressions(currentMetrics, baselineMetrics)
	alertCtx.Suggestions = e.filterSuggestions(suggestions)

	// Generate summary
	alertCtx.Summary = e.generateSummary(alertCtx, len(currentMetrics))

	return alertCtx, nil
}

// analyzeSlowSQL identifies the top N slow queries.
func (e *Engine) analyzeSlowSQL(metrics []model.MetricSnapshot) []model.MetricSnapshot {
	if len(metrics) == 0 {
		return nil
	}

	// Make a copy to avoid modifying the original slice
	sortedMetrics := make([]model.MetricSnapshot, len(metrics))
	copy(sortedMetrics, metrics)

	// Sort based on configured ranking metric
	rankBy := e.cfg.Rules.SlowSQL.RankBy
	sortMetrics(sortedMetrics, rankBy)

	// Take top N
	topN := e.cfg.Rules.SlowSQL.TopN
	if topN > len(sortedMetrics) {
		topN = len(sortedMetrics)
	}

	return sortedMetrics[:topN]
}

// detectRegressions identifies queries with significant performance degradation.
func (e *Engine) detectRegressions(current, baseline []model.MetricSnapshot) []model.RegressionItem {
	if len(current) == 0 || len(baseline) == 0 {
		return nil
	}

	// Build baseline lookup map
	// Use a composite key to handle same queryID across different servers/databases
	type comparisonKey struct {
		queryID int64
		server  string
		db      string
	}

	baselineMap := make(map[comparisonKey]model.MetricSnapshot)
	for _, m := range baseline {
		key := comparisonKey{
			queryID: m.QueryID,
			server:  m.ServerName,
			db:      m.DatabaseName,
		}
		baselineMap[key] = m
	}

	threshold := e.cfg.Rules.Regression.ThresholdPercent
	var regressions []model.RegressionItem

	for _, curr := range current {
		key := comparisonKey{
			queryID: curr.QueryID,
			server:  curr.ServerName,
			db:      curr.DatabaseName,
		}
		base, exists := baselineMap[key]
		if !exists || base.MeanTime == 0 {
			continue
		}

		changePercent := ((curr.MeanTime - base.MeanTime) / base.MeanTime) * 100

		if changePercent >= threshold {
			regressions = append(regressions, model.RegressionItem{
				QueryID:          curr.QueryID,
				Query:            curr.Query,
				DatabaseName:     curr.DatabaseName,
				ServerName:       curr.ServerName,
				CurrentMeanTime:  curr.MeanTime,
				BaselineMeanTime: base.MeanTime,
				ChangePercent:    changePercent,
				CurrentCalls:     curr.Calls,
				BaselineCalls:    base.Calls,
				Severity:         calculateSeverity(changePercent),
			})
		}
	}

	// Sort by change percent descending
	sort.Slice(regressions, func(i, j int) bool {
		return regressions[i].ChangePercent > regressions[j].ChangePercent
	})

	return regressions
}

// filterSuggestions filters index suggestions by minimum improvement threshold.
func (e *Engine) filterSuggestions(suggestions []model.IndexSuggestion) []model.IndexSuggestion {
	if len(suggestions) == 0 {
		return nil
	}

	minImprovement := e.cfg.Rules.IndexSuggestion.MinImprovementPercent
	var filtered []model.IndexSuggestion

	for _, s := range suggestions {
		if s.EstImprovementPercent >= minImprovement {
			filtered = append(filtered, s)
		}
	}

	// Sort by improvement descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].EstImprovementPercent > filtered[j].EstImprovementPercent
	})

	return filtered
}

// generateSummary creates an overall health summary.
func (e *Engine) generateSummary(alertCtx *model.AlertContext, totalQueries int) model.AlertSummary {
	summary := model.AlertSummary{
		TotalQueriesAnalyzed: totalQueries,
		SlowQueryCount:       len(alertCtx.TopSlowSQL),
		RegressionCount:      len(alertCtx.Regressions),
		SuggestionCount:      len(alertCtx.Suggestions),
	}

	// Calculate health score (0-100)
	// Deduct points for issues with upper limits per category
	score := 100

	// Deduct for regressions (more severe) with cap
	regressionDeduction := 0
	for _, r := range alertCtx.Regressions {
		switch r.Severity {
		case "critical":
			regressionDeduction += 20
		case "high":
			regressionDeduction += 10
		case "medium":
			regressionDeduction += 5
		default:
			regressionDeduction += 2
		}
	}
	if regressionDeduction > MaxRegressionDeduction {
		regressionDeduction = MaxRegressionDeduction
	}
	score -= regressionDeduction

	// Deduct for missing indexes with cap
	suggestionDeduction := len(alertCtx.Suggestions) * 3
	if suggestionDeduction > MaxSuggestionDeduction {
		suggestionDeduction = MaxSuggestionDeduction
	}
	score -= suggestionDeduction

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}

	summary.HealthScore = score
	summary.HealthStatus = getHealthStatus(score)

	return summary
}

// sortMetrics sorts metrics by the specified field.
func sortMetrics(metrics []model.MetricSnapshot, rankBy string) {
	sort.Slice(metrics, func(i, j int) bool {
		switch rankBy {
		case "mean_time":
			return metrics[i].MeanTime > metrics[j].MeanTime
		case "cpu_time":
			return metrics[i].TotalCPUTime() > metrics[j].TotalCPUTime()
		case "io_time":
			return metrics[i].IOTime() > metrics[j].IOTime()
		default: // total_time
			return metrics[i].TotalTime > metrics[j].TotalTime
		}
	})
}

// calculateSeverity determines regression severity based on change percent.
func calculateSeverity(changePercent float64) string {
	switch {
	case changePercent >= 500:
		return "critical"
	case changePercent >= 200:
		return "high"
	case changePercent >= 100:
		return "medium"
	default:
		return "low"
	}
}

// getHealthStatus converts a health score to a status string.
func getHealthStatus(score int) string {
	switch {
	case score >= 90:
		return "healthy"
	case score >= 70:
		return "warning"
	case score >= 50:
		return "degraded"
	default:
		return "critical"
	}
}

// generateReqID creates a unique request ID for tracking.
func generateReqID() string {
	return fmt.Sprintf("powa-%d", time.Now().UnixNano())
}
