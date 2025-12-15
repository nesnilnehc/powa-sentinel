// Package model defines the core data structures used by powa-sentinel.
package model

import "time"

// MetricSnapshot represents a point-in-time snapshot of query performance metrics.
// It maps to data from powa_statements and optionally pg_stat_kcache.
type MetricSnapshot struct {
	// QueryID is the unique identifier for the normalized query text (from pg_stat_statements).
	QueryID int64 `json:"query_id"`

	// Query is the normalized query text with placeholders instead of literal values.
	Query string `json:"query"`

	// DatabaseName is the name of the database where the query was executed.
	DatabaseName string `json:"database_name"`

	// TotalTime is the total time spent executing the query in milliseconds.
	TotalTime float64 `json:"total_time"`

	// MeanTime is the average execution time per call in milliseconds.
	MeanTime float64 `json:"mean_time"`

	// Calls is the number of times the query was executed.
	Calls int64 `json:"calls"`

	// Timestamp is the time of the snapshot/aggregation window.
	Timestamp time.Time `json:"timestamp"`

	// --- Optional fields from pg_stat_kcache ---

	// ReadsBlks is the number of blocks read from disk (pg_stat_kcache).
	ReadsBlks int64 `json:"reads_blks,omitempty"`

	// WritesBlks is the number of blocks written to disk (pg_stat_kcache).
	WritesBlks int64 `json:"writes_blks,omitempty"`

	// UserCPUTime is the CPU time spent in user mode in milliseconds (pg_stat_kcache).
	UserCPUTime float64 `json:"user_cpu_time,omitempty"`

	// SystemCPUTime is the CPU time spent in kernel mode in milliseconds (pg_stat_kcache).
	SystemCPUTime float64 `json:"system_cpu_time,omitempty"`

	// HasKCacheData indicates if pg_stat_kcache data is available for this snapshot.
	HasKCacheData bool `json:"has_kcache_data,omitempty"`
}

// TotalCPUTime returns the combined user and system CPU time.
func (m *MetricSnapshot) TotalCPUTime() float64 {
	return m.UserCPUTime + m.SystemCPUTime
}

// IOTime returns a combined I/O metric based on read/write blocks.
// This is a simplified metric; actual I/O time would require more context.
func (m *MetricSnapshot) IOTime() float64 {
	// Approximate I/O time based on block operations
	// This is a placeholder calculation
	return float64(m.ReadsBlks+m.WritesBlks) * 0.01
}
