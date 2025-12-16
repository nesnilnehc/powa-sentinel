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
			// Search for a table matching powa_%kcache%history (e.g. powa_kcache_history or powa_kcache_metrics_history)
			// We prefer 'powa_kcache_history' if both exist, so we order by name length
			var tableName string
			err := r.db.QueryRowContext(ctx, `
				SELECT tablename 
				FROM pg_tables 
				WHERE schemaname = 'public' 
				AND tablename LIKE 'powa_%kcache%history'
				ORDER BY length(tablename) ASC 
				LIMIT 1
			`).Scan(&tableName)

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
				r.kcacheTable = tableName
				log.Printf("Detected PoWA 4 kcache table: %s", tableName)
			}
		} else if r.hasKCache && !r.isPoWA4() {
			// Default for PoWA 3
			r.kcacheTable = "powa_kcache_metrics_history"
		}

		log.Printf("Extension check: pg_stat_kcache=%v (table=%s), pg_qualstats=%v, powa_version=%s", r.hasKCache, r.kcacheTable, r.hasQualStats, r.powaVersion)
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
func (r *Reader) getMetrics(ctx context.Context, startTime, endTime time.Time) ([]model.MetricSnapshot, error) {
	// Use LIMIT to prevent unbounded result sets
	var query string

	if r.isPoWA4() {
		// PoWA 4 uses a nested "records" array and queryid/dbid/srvid structure
		// We unnest the records to get the individual metrics
		// Note: PoWA 4 records typically use standardized field names like total_exec_time
		// We also join with powa_servers to get the server alias/name
		query = fmt.Sprintf(`
			SELECT 
				ps.queryid,
				s.query,
				pd.datname,
				COALESCE(srv.alias, srv.hostname || ':' || CAST(srv.port AS TEXT)) as server_name,
				ps.srvid,
				COALESCE(SUM((r).total_exec_time), 0) as total_time,
				COALESCE(SUM((r).total_exec_time) / NULLIF(SUM((r).calls), 0), 0) as mean_time,
				COALESCE(SUM((r).calls), 0) as calls,
				MAX((r).ts) as ts
			FROM powa_statements_history ps
			CROSS JOIN LATERAL unnest(ps.records) as r
			JOIN powa_databases pd ON ps.srvid = pd.srvid AND ps.dbid = pd.oid
			JOIN powa_statements s ON ps.srvid = s.srvid AND ps.queryid = s.queryid AND ps.dbid = s.dbid AND ps.userid = s.userid
			JOIN powa_servers srv ON ps.srvid = srv.id
			WHERE ps.coalesce_range && tstzrange($1::timestamptz, $2::timestamptz, '[]')
			AND (r).ts >= $1 AND (r).ts <= $2
			GROUP BY ps.queryid, s.query, pd.datname, server_name, ps.srvid
			ORDER BY total_time DESC
			LIMIT %d
		`, MaxQueryRows)
	} else {
		// PoWA 3 uses flat columns
		// Dynamically select the correct column name based on PostgreSQL version
		execTimeCol := r.getExecTimeColumn()
		query = fmt.Sprintf(`
			SELECT 
				ps.queryid,
				s.query,
				pd.datname,
				'local' as server_name,
				0 as srvid,
				COALESCE(SUM(ps.%s), 0) as total_time,
				COALESCE(SUM(ps.%s) / NULLIF(SUM(ps.calls), 0), 0) as mean_time,
				COALESCE(SUM(ps.calls), 0) as calls,
				MAX(ps.ts) as ts
			FROM powa_statements_history ps
			JOIN powa_databases pd ON ps.dbid = pd.oid
			JOIN powa_statements s ON ps.queryid = s.queryid AND ps.dbid = s.dbid AND ps.userid = s.userid
			WHERE ps.ts >= $1 AND ps.ts <= $2
			GROUP BY ps.queryid, s.query, pd.datname
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
func (r *Reader) enrichWithKCache(ctx context.Context, snapshots []model.MetricSnapshot, startTime, endTime time.Time) error {
	var query string
	if r.isPoWA4() {
		// PoWA 4 schema - using discovered table
		query = fmt.Sprintf(`
			SELECT 
				k.queryid,
				k.srvid,
				COALESCE(SUM((r).exec_reads), 0) as reads_blks,
				COALESCE(SUM((r).exec_writes), 0) as writes_blks,
				COALESCE(SUM((r).exec_user_time), 0) as user_cpu_time,
				COALESCE(SUM((r).exec_system_time), 0) as system_cpu_time
			FROM %s k
			CROSS JOIN LATERAL unnest(k.records) as r
			WHERE k.coalesce_range && tstzrange($1::timestamptz, $2::timestamptz, '[]')
			AND (r).ts >= $1 AND (r).ts <= $2
			GROUP BY k.queryid, k.srvid
		`, r.kcacheTable)
	} else {
		// PoWA 3 schema (legacy)
		query = fmt.Sprintf(`
			SELECT 
				queryid,
				0 as srvid,
				COALESCE(SUM(reads), 0) as reads_blks,
				COALESCE(SUM(writes), 0) as writes_blks,
				COALESCE(SUM(user_time), 0) as user_cpu_time,
				COALESCE(SUM(system_time), 0) as system_cpu_time
			FROM %s
			WHERE ts >= $1 AND ts <= $2
			GROUP BY queryid
		`, r.kcacheTable)
	}

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		// Don't fail the whole analysis if kcache enrichment fails
		fmt.Printf("Warning: failed to enrich with kcache data: %v\n", err)
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
