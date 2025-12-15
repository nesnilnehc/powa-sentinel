package model

import "time"

// AlertContext contains all the analysis results to be included in a notification.
type AlertContext struct {
	// ReqID is a unique identifier for this alert/analysis run.
	ReqID string `json:"req_id"`

	// ReportType describes the type of report (e.g., "weekly", "daily", "adhoc").
	ReportType string `json:"report_type"`

	// Timestamp is when this alert was generated.
	Timestamp time.Time `json:"timestamp"`

	// AnalysisWindow describes the time window analyzed.
	AnalysisWindow TimeWindow `json:"analysis_window"`

	// BaselineWindow describes the comparison baseline time window.
	BaselineWindow TimeWindow `json:"baseline_window"`

	// DatabaseName is the target database being analyzed.
	DatabaseName string `json:"database_name"`

	// TopSlowSQL contains the top N slow queries identified.
	TopSlowSQL []MetricSnapshot `json:"top_slow_sql,omitempty"`

	// Regressions contains queries with significant performance degradation.
	Regressions []RegressionItem `json:"regressions,omitempty"`

	// Suggestions contains index optimization recommendations.
	Suggestions []IndexSuggestion `json:"suggestions,omitempty"`

	// Summary contains aggregated health metrics.
	Summary AlertSummary `json:"summary"`
}

// TimeWindow represents a time range for analysis.
type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Duration returns the duration of the time window.
func (w TimeWindow) Duration() time.Duration {
	return w.End.Sub(w.Start)
}

// AlertSummary provides high-level health indicators.
type AlertSummary struct {
	// TotalQueriesAnalyzed is the count of unique queries in the analysis window.
	TotalQueriesAnalyzed int `json:"total_queries_analyzed"`

	// SlowQueryCount is the number of queries exceeding thresholds.
	SlowQueryCount int `json:"slow_query_count"`

	// RegressionCount is the number of detected performance regressions.
	RegressionCount int `json:"regression_count"`

	// SuggestionCount is the number of index optimization suggestions.
	SuggestionCount int `json:"suggestion_count"`

	// HealthScore is an overall health score from 0-100.
	HealthScore int `json:"health_score"`

	// HealthStatus is a human-readable status (e.g., "healthy", "warning", "critical").
	HealthStatus string `json:"health_status"`
}

// RegressionItem represents a query with detected performance regression.
type RegressionItem struct {
	// QueryID is the unique identifier for the query.
	QueryID int64 `json:"query_id"`

	// Query is the normalized query text.
	Query string `json:"query"`

	// DatabaseName is the database where the query runs.
	DatabaseName string `json:"database_name"`

	// CurrentMeanTime is the mean execution time in the current window.
	CurrentMeanTime float64 `json:"current_mean_time"`

	// BaselineMeanTime is the mean execution time in the baseline window.
	BaselineMeanTime float64 `json:"baseline_mean_time"`

	// ChangePercent is the percentage change ((current - baseline) / baseline * 100).
	ChangePercent float64 `json:"change_percent"`

	// CurrentCalls is the number of calls in the current window.
	CurrentCalls int64 `json:"current_calls"`

	// BaselineCalls is the number of calls in the baseline window.
	BaselineCalls int64 `json:"baseline_calls"`

	// Severity indicates the regression severity ("low", "medium", "high", "critical").
	Severity string `json:"severity"`
}

// IndexSuggestion represents a missing index recommendation.
type IndexSuggestion struct {
	// Table is the table name that would benefit from an index.
	Table string `json:"table"`

	// Schema is the schema name containing the table.
	Schema string `json:"schema"`

	// Columns are the column names suggested for the index.
	Columns []string `json:"columns"`

	// AccessType describes the current access pattern (e.g., "Seq Scan").
	AccessType string `json:"access_type"`

	// QualType describes the predicate type (e.g., "equality", "range").
	QualType string `json:"qual_type"`

	// EstImprovementPercent is the estimated performance improvement percentage.
	EstImprovementPercent float64 `json:"est_improvement_percent"`

	// AffectedQueries is the count of queries that would benefit from this index.
	AffectedQueries int `json:"affected_queries"`

	// SuggestedDDL is the CREATE INDEX statement (if available from hypopg).
	SuggestedDDL string `json:"suggested_ddl,omitempty"`
}

// FullTableName returns the fully qualified table name.
func (s *IndexSuggestion) FullTableName() string {
	if s.Schema == "" || s.Schema == "public" {
		return s.Table
	}
	return s.Schema + "." + s.Table
}
