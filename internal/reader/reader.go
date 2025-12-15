// Package reader provides database access for reading PoWA performance data.
package reader

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/powa-team/powa-sentinel/internal/config"
	"github.com/powa-team/powa-sentinel/internal/model"
)

// MaxQueryRows limits the number of rows returned by metrics queries.
const MaxQueryRows = 10000

// Reader handles database connections and queries to the PoWA repository.
type Reader struct {
	db           *sql.DB
	cfg          *config.DatabaseConfig
	hasKCache    bool
	hasQualStats bool

	// extensionsOnce ensures extension check runs only once
	extensionsOnce sync.Once
	extensionsErr  error
}

// New creates a new Reader with the given database configuration.
func New(cfg *config.DatabaseConfig) (*Reader, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	reader := &Reader{
		db:  db,
		cfg: cfg,
	}

	return reader, nil
}

// Ping tests the database connection.
func (r *Reader) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// Close closes the database connection.
func (r *Reader) Close() error {
	return r.db.Close()
}

// checkExtensions checks for optional extensions (pg_stat_kcache, pg_qualstats).
// Thread-safe: uses sync.Once to ensure it only runs once.
func (r *Reader) checkExtensions(ctx context.Context) error {
	r.extensionsOnce.Do(func() {
		// Check for pg_stat_kcache
		var hasKCache bool
		err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_kcache'
			)
		`).Scan(&hasKCache)
		if err != nil {
			r.extensionsErr = fmt.Errorf("checking pg_stat_kcache: %w", err)
			return
		}
		r.hasKCache = hasKCache

		// Check for pg_qualstats
		var hasQualStats bool
		err = r.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_extension WHERE extname = 'pg_qualstats'
			)
		`).Scan(&hasQualStats)
		if err != nil {
			r.extensionsErr = fmt.Errorf("checking pg_qualstats: %w", err)
			return
		}
		r.hasQualStats = hasQualStats

		log.Printf("Extension check: pg_stat_kcache=%v, pg_qualstats=%v", hasKCache, hasQualStats)
	})

	return r.extensionsErr
}

// HasKCache returns whether pg_stat_kcache is available.
func (r *Reader) HasKCache() bool {
	return r.hasKCache
}

// HasQualStats returns whether pg_qualstats is available.
func (r *Reader) HasQualStats() bool {
	return r.hasQualStats
}

// GetCurrentMetrics fetches performance metrics for the specified time window.
func (r *Reader) GetCurrentMetrics(ctx context.Context, window time.Duration) ([]model.MetricSnapshot, error) {
	if err := r.checkExtensions(ctx); err != nil {
		return nil, err
	}

	endTime := time.Now()
	startTime := endTime.Add(-window)

	return r.getMetrics(ctx, startTime, endTime)
}

// GetBaselineMetrics fetches baseline metrics for comparison.
func (r *Reader) GetBaselineMetrics(ctx context.Context, offset, window time.Duration) ([]model.MetricSnapshot, error) {
	if err := r.checkExtensions(ctx); err != nil {
		return nil, err
	}

	endTime := time.Now().Add(-offset)
	startTime := endTime.Add(-window)

	return r.getMetrics(ctx, startTime, endTime)
}

// getMetrics fetches metrics for a specific time range.
func (r *Reader) getMetrics(ctx context.Context, startTime, endTime time.Time) ([]model.MetricSnapshot, error) {
	// Use LIMIT to prevent unbounded result sets
	query := fmt.Sprintf(`
		SELECT 
			ps.queryid,
			ps.query,
			pd.datname,
			COALESCE(SUM(ps.total_exec_time), 0) as total_time,
			COALESCE(AVG(ps.mean_exec_time), 0) as mean_time,
			COALESCE(SUM(ps.calls), 0) as calls,
			MAX(ps.ts) as ts
		FROM powa_statements_history ps
		JOIN powa_databases pd ON ps.dbid = pd.oid
		WHERE ps.ts >= $1 AND ps.ts <= $2
		GROUP BY ps.queryid, ps.query, pd.datname
		ORDER BY total_time DESC
		LIMIT %d
	`, MaxQueryRows)

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("querying powa_statements_history: %w", err)
	}
	defer rows.Close()

	var snapshots []model.MetricSnapshot
	for rows.Next() {
		var s model.MetricSnapshot
		err := rows.Scan(
			&s.QueryID,
			&s.Query,
			&s.DatabaseName,
			&s.TotalTime,
			&s.MeanTime,
			&s.Calls,
			&s.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		snapshots = append(snapshots, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	// If pg_stat_kcache is available, enrich with CPU/IO data
	if r.hasKCache && len(snapshots) > 0 {
		if err := r.enrichWithKCache(ctx, snapshots, startTime, endTime); err != nil {
			// Log warning but don't fail - kcache data is optional
			log.Printf("Warning: failed to enrich with kcache data: %v", err)
		}
	}

	return snapshots, nil
}

// enrichWithKCache adds pg_stat_kcache metrics to the snapshots.
func (r *Reader) enrichWithKCache(ctx context.Context, snapshots []model.MetricSnapshot, startTime, endTime time.Time) error {
	query := `
		SELECT 
			queryid,
			COALESCE(SUM(reads), 0) as reads_blks,
			COALESCE(SUM(writes), 0) as writes_blks,
			COALESCE(SUM(user_time), 0) as user_cpu_time,
			COALESCE(SUM(system_time), 0) as system_cpu_time
		FROM powa_kcache_metrics_history
		WHERE ts >= $1 AND ts <= $2
		GROUP BY queryid
	`

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return fmt.Errorf("querying powa_kcache_metrics_history: %w", err)
	}
	defer rows.Close()

	// Build a map for quick lookup
	kcacheData := make(map[int64]struct {
		ReadsBlks     int64
		WritesBlks    int64
		UserCPUTime   float64
		SystemCPUTime float64
	})

	for rows.Next() {
		var queryID int64
		var reads, writes int64
		var userCPU, sysCPU float64
		if err := rows.Scan(&queryID, &reads, &writes, &userCPU, &sysCPU); err != nil {
			return fmt.Errorf("scanning kcache row: %w", err)
		}
		kcacheData[queryID] = struct {
			ReadsBlks     int64
			WritesBlks    int64
			UserCPUTime   float64
			SystemCPUTime float64
		}{reads, writes, userCPU, sysCPU}
	}

	// Enrich snapshots
	for i := range snapshots {
		if kc, ok := kcacheData[snapshots[i].QueryID]; ok {
			snapshots[i].ReadsBlks = kc.ReadsBlks
			snapshots[i].WritesBlks = kc.WritesBlks
			snapshots[i].UserCPUTime = kc.UserCPUTime
			snapshots[i].SystemCPUTime = kc.SystemCPUTime
			snapshots[i].HasKCacheData = true
		}
	}

	return nil
}

// isViewNotExistError checks if the error is due to a missing view/table.
func isViewNotExistError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// 42P01 = undefined_table
		return pqErr.Code == "42P01"
	}
	return false
}

// isPermissionError checks if the error is due to permission denied.
func isPermissionError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// 42501 = insufficient_privilege
		return pqErr.Code == "42501"
	}
	return false
}

// GetIndexSuggestions fetches missing index suggestions from pg_qualstats.
func (r *Reader) GetIndexSuggestions(ctx context.Context) ([]model.IndexSuggestion, error) {
	if err := r.checkExtensions(ctx); err != nil {
		return nil, err
	}

	if !r.hasQualStats {
		return nil, nil // No suggestions available without pg_qualstats
	}

	// Query the powa_qualstats view for index suggestions
	// This is a simplified query; actual implementation may vary based on PoWA version
	query := `
		SELECT 
			relname as table_name,
			nspname as schema_name,
			array_agg(DISTINCT attname) as columns,
			qualtype,
			avg_filter as est_improvement_percent,
			count(*) as affected_queries
		FROM powa_qualstats_indexes
		WHERE suggestion IS NOT NULL
		GROUP BY relname, nspname, qualtype, avg_filter
		ORDER BY est_improvement_percent DESC
		LIMIT 100
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		// Handle expected errors gracefully
		if isViewNotExistError(err) {
			log.Printf("Warning: powa_qualstats_indexes view does not exist, skipping index suggestions")
			return nil, nil
		}
		if isPermissionError(err) {
			log.Printf("Warning: insufficient privileges to query powa_qualstats_indexes, skipping index suggestions")
			return nil, nil
		}
		// Unexpected error - log and return
		log.Printf("Error querying powa_qualstats_indexes: %v", err)
		return nil, fmt.Errorf("querying powa_qualstats_indexes: %w", err)
	}
	defer rows.Close()

	var suggestions []model.IndexSuggestion
	var scanErrors int
	for rows.Next() {
		var s model.IndexSuggestion
		var columns []string
		err := rows.Scan(
			&s.Table,
			&s.Schema,
			pq.Array(&columns),
			&s.QualType,
			&s.EstImprovementPercent,
			&s.AffectedQueries,
		)
		if err != nil {
			scanErrors++
			if scanErrors <= 3 {
				log.Printf("Warning: failed to scan index suggestion row: %v", err)
			}
			continue // Skip malformed rows
		}
		s.Columns = columns
		s.AccessType = "Seq Scan" // Default; could be refined
		suggestions = append(suggestions, s)
	}

	if scanErrors > 3 {
		log.Printf("Warning: %d total rows failed to scan in GetIndexSuggestions", scanErrors)
	}

	return suggestions, nil
}

// GetDatabaseList returns the list of databases in the PoWA repository.
func (r *Reader) GetDatabaseList(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT datname FROM powa_databases ORDER BY datname`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying powa_databases: %w", err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning database name: %w", err)
		}
		databases = append(databases, name)
	}

	return databases, nil
}
