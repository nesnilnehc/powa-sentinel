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
	pgVersion    int    // e.g. 140000
	powaVersion  string // e.g. 4.0.1
	kcacheTable  string // Detected table name for kcache history

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
		// Detect PostgreSQL version
		var versionNum int
		err := r.db.QueryRowContext(ctx, "SHOW server_version_num").Scan(&versionNum)
		if err != nil {
			r.extensionsErr = fmt.Errorf("detecting PostgreSQL version: %w", err)
			return
		}
		r.pgVersion = versionNum

		// Detect PoWA version
		var powaVersion string
		err = r.db.QueryRowContext(ctx, "SELECT extversion FROM pg_extension WHERE extname = 'powa'").Scan(&powaVersion)
		if err != nil {
			// If powa extension is missing, we can't do anything
			r.extensionsErr = fmt.Errorf("detecting PoWA version: %w", err)
			return
		}
		r.powaVersion = powaVersion

		// Check for pg_stat_kcache
		var hasKCache bool
		err = r.db.QueryRowContext(ctx, `
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
		err = r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_qualstats')").Scan(&hasQualStats)
		if err != nil {
			r.extensionsErr = fmt.Errorf("checking pg_qualstats extension: %w", err)
			return
		}
		r.hasQualStats = hasQualStats

		// If PoWA 4+ and kcache is enabled, try to find the correct history table
		if r.hasKCache && r.isPoWA4() {
			// Search for a table matching powa_%kcache%history in both public and powa schemas
			// (PoWA archivist typically creates tables in the powa schema)
			var schemaName, tableName string
			err := r.db.QueryRowContext(ctx, `
				SELECT schemaname, tablename 
				FROM pg_tables 
				WHERE schemaname IN ('public', 'powa') 
				AND tablename LIKE 'powa_%kcache%history'
				ORDER BY length(tablename) ASC 
				LIMIT 1
			`).Scan(&schemaName, &tableName)

			if err != nil {
				if err == sql.ErrNoRows {
					log.Printf("Warning: pg_stat_kcache extension present but no history table found in PoWA 4. Disabling kcache enrichment.")
					r.hasKCache = false
				} else {
					// Don't fail completely, just log
					log.Printf("Warning: error searching for kcache table: %v. Disabling kcache.", err)
					r.hasKCache = false
				}
			} else {
				r.kcacheTable = schemaName + "." + tableName
				log.Printf("Detected PoWA 4 kcache table: %s", r.kcacheTable)
			}
		} else if r.hasKCache && !r.isPoWA4() {
			// Default for PoWA 3
			r.kcacheTable = "powa_kcache_metrics_history"
		}

		log.Printf("Extension check: pg_stat_kcache=%v (table=%s), pg_qualstats=%v, powa_version=%s", r.hasKCache, r.kcacheTable, r.hasQualStats, r.powaVersion)

		// Optional environment expectation check: compare expected_extensions with actual availability
		if len(r.cfg.ExpectedExtensions) > 0 {
			actual := make(map[string]bool)
			if r.hasKCache {
				actual["pg_stat_kcache"] = true
			}
			if r.hasQualStats {
				actual["pg_qualstats"] = true
			}
			seenMissing := make(map[string]bool)
			var missing []string
			for _, ext := range r.cfg.ExpectedExtensions {
				if !actual[ext] && !seenMissing[ext] {
					seenMissing[ext] = true
					missing = append(missing, ext)
				}
			}
			if len(missing) > 0 {
				log.Printf("Environment check: expected extensions %v; missing: %v", r.cfg.ExpectedExtensions, missing)
			}
		}
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

// getExecTimeColumn returns the correct column name for execution time based on PostgreSQL version.
// PostgreSQL 13+ uses "total_exec_time", earlier versions use "total_time".
func (r *Reader) getExecTimeColumn() string {
	if r.pgVersion >= 130000 {
		return "total_exec_time"
	}
	return "total_time"
}

// isPoWA4 returns true if the detected PoWA version is 4.x or higher.
func (r *Reader) isPoWA4() bool {
	return len(r.powaVersion) > 0 && r.powaVersion[0] >= '4'
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
//
// PoWA history tables store cumulative counters (calls, total_exec_time, etc.). For a time window,
// we must compute the delta (last − first) per (queryid, …), not SUM of rows. getMetrics and
// enrichWithKCache both use this first/last aggregation pattern; see powa-schema.md for schema notes.
func (r *Reader) getMetrics(ctx context.Context, startTime, endTime time.Time) ([]model.MetricSnapshot, error) {
	// Use LIMIT to prevent unbounded result sets
	var query string

	if r.isPoWA4() {
		// PoWA 4 uses a nested "records" array; each record holds cumulative stats at that ts.
		// We must use delta (last - first) in the window, not SUM, to get calls/total_time for the period.
		query = fmt.Sprintf(`
			WITH u AS (
				SELECT ps.queryid, ps.srvid, ps.dbid, ps.userid,
					(r).ts AS ts,
					(r).calls AS calls,
					(r).total_exec_time AS total_exec_time
				FROM powa_statements_history ps
				CROSS JOIN LATERAL unnest(ps.records) AS r
				WHERE ps.coalesce_range && tstzrange($1::timestamptz, $2::timestamptz, '[]')
					AND (r).ts >= $1 AND (r).ts <= $2
			),
			first_last AS (
				SELECT
					queryid, srvid, dbid, userid,
					(array_agg(calls ORDER BY ts))[1] AS first_calls,
					(array_agg(total_exec_time ORDER BY ts))[1] AS first_time,
					(array_agg(calls ORDER BY ts DESC))[1] AS last_calls,
					(array_agg(total_exec_time ORDER BY ts DESC))[1] AS last_time,
					MAX(ts) AS ts
				FROM u
				GROUP BY queryid, srvid, dbid, userid
			)
			SELECT
				fl.queryid,
				s.query,
				pd.datname,
				COALESCE(srv.alias, srv.hostname || ':' || CAST(srv.port AS TEXT)) AS server_name,
				fl.srvid,
				COALESCE(GREATEST(fl.last_time - fl.first_time, 0), 0) AS total_time,
				CASE WHEN (fl.last_calls - fl.first_calls) > 0
					THEN (fl.last_time - fl.first_time) / NULLIF(fl.last_calls - fl.first_calls, 0)
					ELSE 0 END AS mean_time,
				COALESCE(GREATEST(fl.last_calls - fl.first_calls, 0), 0)::bigint AS calls,
				fl.ts
			FROM first_last fl
			JOIN powa_databases pd ON fl.srvid = pd.srvid AND fl.dbid = pd.oid
			JOIN powa_statements s ON fl.srvid = s.srvid AND fl.queryid = s.queryid AND fl.dbid = s.dbid AND fl.userid = s.userid
			JOIN powa_servers srv ON fl.srvid = srv.id
			ORDER BY total_time DESC
			LIMIT %d
		`, MaxQueryRows)
	} else {
		// PoWA 3 uses flat columns; each row is a cumulative snapshot at ps.ts.
		// Use delta (last - first) in the window, not SUM, to get calls/total_time for the period.
		execTimeCol := r.getExecTimeColumn()
		query = fmt.Sprintf(`
			WITH first_last AS (
				SELECT
					ps.queryid, ps.dbid, ps.userid,
					(array_agg(ps.calls ORDER BY ps.ts))[1] AS first_calls,
					(array_agg(ps.%s ORDER BY ps.ts))[1] AS first_time,
					(array_agg(ps.calls ORDER BY ps.ts DESC))[1] AS last_calls,
					(array_agg(ps.%s ORDER BY ps.ts DESC))[1] AS last_time,
					MAX(ps.ts) AS ts
				FROM powa_statements_history ps
				WHERE ps.ts >= $1 AND ps.ts <= $2
				GROUP BY ps.queryid, ps.dbid, ps.userid
			)
			SELECT
				fl.queryid,
				s.query,
				pd.datname,
				'local' AS server_name,
				0 AS srvid,
				COALESCE(GREATEST(fl.last_time - fl.first_time, 0), 0) AS total_time,
				CASE WHEN (fl.last_calls - fl.first_calls) > 0
					THEN (fl.last_time - fl.first_time) / NULLIF(fl.last_calls - fl.first_calls, 0)
					ELSE 0 END AS mean_time,
				COALESCE(GREATEST(fl.last_calls - fl.first_calls, 0), 0)::bigint AS calls,
				fl.ts
			FROM first_last fl
			JOIN powa_databases pd ON fl.dbid = pd.oid
			JOIN powa_statements s ON fl.queryid = s.queryid AND fl.dbid = s.dbid AND fl.userid = s.userid
			ORDER BY total_time DESC
			LIMIT %d
		`, execTimeCol, execTimeCol, MaxQueryRows)
	}

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("querying powa_statements_history: %w", err)
	}
	defer rows.Close()

	var snapshots []model.MetricSnapshot
	for rows.Next() {
		var m model.MetricSnapshot
		if err := rows.Scan(
			&m.QueryID,
			&m.Query,
			&m.DatabaseName,
			&m.ServerName,
			&m.SrvID,
			&m.TotalTime,
			&m.MeanTime,
			&m.Calls,
			&m.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("scanning metrics row: %w", err)
		}
		snapshots = append(snapshots, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating metrics rows: %w", err)
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
// Kcache history stores cumulative counters; we use delta (last - first) in the window, not SUM.
func (r *Reader) enrichWithKCache(ctx context.Context, snapshots []model.MetricSnapshot, startTime, endTime time.Time) error {
	var query string
	if r.isPoWA4() {
		// PoWA 4: first/last delta per (queryid, srvid)
		query = fmt.Sprintf(`
			WITH u AS (
				SELECT k.queryid, k.srvid,
					(r).ts AS ts,
					(r).exec_reads AS exec_reads,
					(r).exec_writes AS exec_writes,
					(r).exec_user_time AS exec_user_time,
					(r).exec_system_time AS exec_system_time
				FROM %s k
				CROSS JOIN LATERAL unnest(k.records) AS r
				WHERE k.coalesce_range && tstzrange($1::timestamptz, $2::timestamptz, '[]')
					AND (r).ts >= $1 AND (r).ts <= $2
			),
			first_last AS (
				SELECT
					queryid, srvid,
					(array_agg(exec_reads ORDER BY ts))[1] AS first_reads,
					(array_agg(exec_writes ORDER BY ts))[1] AS first_writes,
					(array_agg(exec_user_time ORDER BY ts))[1] AS first_user_time,
					(array_agg(exec_system_time ORDER BY ts))[1] AS first_system_time,
					(array_agg(exec_reads ORDER BY ts DESC))[1] AS last_reads,
					(array_agg(exec_writes ORDER BY ts DESC))[1] AS last_writes,
					(array_agg(exec_user_time ORDER BY ts DESC))[1] AS last_user_time,
					(array_agg(exec_system_time ORDER BY ts DESC))[1] AS last_system_time
				FROM u
				GROUP BY queryid, srvid
			)
			SELECT
				queryid,
				srvid,
				COALESCE(GREATEST(last_reads - first_reads, 0), 0)::bigint AS reads_blks,
				COALESCE(GREATEST(last_writes - first_writes, 0), 0)::bigint AS writes_blks,
				COALESCE(GREATEST(last_user_time - first_user_time, 0), 0) AS user_cpu_time,
				COALESCE(GREATEST(last_system_time - first_system_time, 0), 0) AS system_cpu_time
			FROM first_last
		`, r.kcacheTable)
	} else {
		// PoWA 3: first/last delta per queryid
		query = fmt.Sprintf(`
			WITH first_last AS (
				SELECT
					queryid,
					(array_agg(reads ORDER BY ts))[1] AS first_reads,
					(array_agg(writes ORDER BY ts))[1] AS first_writes,
					(array_agg(user_time ORDER BY ts))[1] AS first_user_time,
					(array_agg(system_time ORDER BY ts))[1] AS first_system_time,
					(array_agg(reads ORDER BY ts DESC))[1] AS last_reads,
					(array_agg(writes ORDER BY ts DESC))[1] AS last_writes,
					(array_agg(user_time ORDER BY ts DESC))[1] AS last_user_time,
					(array_agg(system_time ORDER BY ts DESC))[1] AS last_system_time
				FROM %s
				WHERE ts >= $1 AND ts <= $2
				GROUP BY queryid
			)
			SELECT
				queryid,
				0 AS srvid,
				COALESCE(GREATEST(last_reads - first_reads, 0), 0)::bigint AS reads_blks,
				COALESCE(GREATEST(last_writes - first_writes, 0), 0)::bigint AS writes_blks,
				COALESCE(GREATEST(last_user_time - first_user_time, 0), 0) AS user_cpu_time,
				COALESCE(GREATEST(last_system_time - first_system_time, 0), 0) AS system_cpu_time
			FROM first_last
		`, r.kcacheTable)
	}

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		// Don't fail the whole analysis if kcache enrichment fails
		log.Printf("Warning: failed to enrich with kcache data: %v", err)
		return nil
	}
	defer rows.Close()

	type kcacheKey struct {
		queryID int64
		srvID   int
	}

	type kcacheData struct {
		reads      int64
		writes     int64
		userTime   float64
		systemTime float64
	}

	kcacheMap := make(map[kcacheKey]kcacheData)
	for rows.Next() {
		var qid int64
		var srvID int
		var data kcacheData
		if err := rows.Scan(&qid, &srvID, &data.reads, &data.writes, &data.userTime, &data.systemTime); err != nil {
			continue
		}
		kcacheMap[kcacheKey{qid, srvID}] = data
	}

	for i := range snapshots {
		m := &snapshots[i]
		if data, ok := kcacheMap[kcacheKey{m.QueryID, m.SrvID}]; ok {
			m.ReadsBlks = data.reads
			m.WritesBlks = data.writes
			m.UserCPUTime = data.userTime
			m.SystemCPUTime = data.systemTime
			m.HasKCacheData = true
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
