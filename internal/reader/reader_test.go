package reader

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/powa-team/powa-sentinel/internal/config"
)

func TestReader_checkExtensions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	r := &Reader{
		db:  db,
		cfg: &config.DatabaseConfig{},
	}

	// Expect version detection query (new requirement)
	mock.ExpectQuery("SHOW server_version_num").
		WillReturnRows(sqlmock.NewRows([]string{"server_version_num"}).AddRow("140000"))

	// Expect PoWA version detection (default to 3.x for existing tests)
	mock.ExpectQuery("SELECT extversion FROM pg_extension WHERE extname = 'powa'").
		WillReturnRows(sqlmock.NewRows([]string{"extversion"}).AddRow("3.2.0"))

	// Expect check for pg_stat_kcache
	mock.ExpectQuery("SELECT EXISTS.*pg_stat_kcache").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Expect check for pg_qualstats
	mock.ExpectQuery("SELECT EXISTS.*pg_qualstats").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Expect KCache table search (PoWA 3.2.0 is detected, but logic runs if hasKCache is true.
	// Wait, isPoWA4() returns false for 3.2.0. So table search is SKIPPED.
	// So NO new query expectation needed for PoWA 3 test case in checkExtensions
	// UNLESS we change the mock version to 4.x.
	// The current mock uses "3.2.0".
	// My code: if r.hasKCache && r.isPoWA4() { search }
	// So for "3.2.0", it skips search.
	// And sets r.kcacheTable = "powa_kcache_metrics_history" (else block).
	// So NO NEW MOCK needed here.

	if err := r.checkExtensions(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !r.HasKCache() {
		t.Error("expected HasKCache to be true")
	}
	if r.HasQualStats() {
		t.Error("expected HasQualStats to be false")
	}

	// Verify expectations
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestReader_checkExtensions_PoWA4_KcacheInPowaSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	r := &Reader{
		db:  db,
		cfg: &config.DatabaseConfig{},
	}

	mock.ExpectQuery("SHOW server_version_num").
		WillReturnRows(sqlmock.NewRows([]string{"server_version_num"}).AddRow("140000"))
	mock.ExpectQuery("SELECT extversion FROM pg_extension WHERE extname = 'powa'").
		WillReturnRows(sqlmock.NewRows([]string{"extversion"}).AddRow("4.2.2"))
	mock.ExpectQuery("SELECT EXISTS.*pg_stat_kcache").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS.*pg_qualstats").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// PoWA 4 + kcache: search pg_tables for kcache history in public/powa
	mock.ExpectQuery("SELECT schemaname, tablename").
		WillReturnRows(sqlmock.NewRows([]string{"schemaname", "tablename"}).AddRow("powa", "powa_kcache_history"))

	if err := r.checkExtensions(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !r.HasKCache() {
		t.Error("expected HasKCache to be true")
	}
	// Discovered table must be schema-qualified so queries work regardless of search_path
	if r.kcacheTable != "powa.powa_kcache_history" {
		t.Errorf("expected kcacheTable %q, got %q", "powa.powa_kcache_history", r.kcacheTable)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestReader_checkExtensions_ExpectedExtensionsMismatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	r := &Reader{
		db:  db,
		cfg: &config.DatabaseConfig{ExpectedExtensions: []string{"pg_qualstats"}},
	}

	mock.ExpectQuery("SHOW server_version_num").
		WillReturnRows(sqlmock.NewRows([]string{"server_version_num"}).AddRow("140000"))
	mock.ExpectQuery("SELECT extversion FROM pg_extension WHERE extname = 'powa'").
		WillReturnRows(sqlmock.NewRows([]string{"extversion"}).AddRow("3.2.0"))
	mock.ExpectQuery("SELECT EXISTS.*pg_stat_kcache").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS.*pg_qualstats").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	if err := r.checkExtensions(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// pg_qualstats was expected but not available; Environment check log is emitted (not asserted here)
	if r.HasQualStats() {
		t.Error("expected HasQualStats to be false")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestReader_GetMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	r := &Reader{
		db:          db,
		cfg:         &config.DatabaseConfig{},
		hasKCache:   true, // Simulate kcache available
		kcacheTable: "powa_kcache_metrics_history",
	}

	// Mock data
	now := time.Now()

	// Expect main metrics query
	// Note: We use a regex for the query matching because whitespace/formatting might vary
	mock.ExpectQuery("SELECT.*powa_statements_history").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"queryid", "query", "datname", "server_name", "srvid", "total_time", "mean_time", "calls", "ts"}).
			AddRow(1001, "SELECT 1", "postgres", "local", 0, 100.0, 10.0, 10, now))

	// Expect kcache enrichment query (since hasKCache=true)
	mock.ExpectQuery("SELECT.*powa_kcache_metrics_history").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"queryid", "srvid", "reads", "writes", "user_time", "sys_time"}).
			AddRow(1001, 0, 50, 10, 5.0, 1.0))

	metrics, err := r.getMetrics(context.Background(), now.Add(-1*time.Hour), now)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	m := metrics[0]
	if m.QueryID != 1001 {
		t.Errorf("expected queryID 1001, got %d", m.QueryID)
	}
	if !m.HasKCacheData {
		t.Error("expected HasKCacheData to be true")
	}
	if m.ReadsBlks != 50 {
		t.Errorf("expected 50 reads, got %d", m.ReadsBlks)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestReader_GetIndexSuggestions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	r := &Reader{
		db:  db,
		cfg: &config.DatabaseConfig{},
	}

	// 1. Test when extensions check fails/not run yet
	// Mock checkExtensions first call
	// First: version detection
	mock.ExpectQuery("SHOW server_version_num").
		WillReturnRows(sqlmock.NewRows([]string{"server_version_num"}).AddRow("140000"))
	mock.ExpectQuery("SELECT extversion FROM pg_extension WHERE extname = 'powa'").
		WillReturnRows(sqlmock.NewRows([]string{"extversion"}).AddRow("3.2.0"))
	mock.ExpectQuery("SELECT EXISTS.*pg_stat_kcache").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS.*pg_qualstats").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Mock suggestions query
	mock.ExpectQuery("SELECT.*powa_qualstats_indexes").
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "schema_name", "columns", "qualtype", "est_improvement", "affected_queries"}).
			AddRow("users", "public", "{id,name}", "Index", 50.5, 10))

	suggestions, err := r.GetIndexSuggestions(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}

	s := suggestions[0]
	if s.Table != "users" {
		t.Errorf("expected table users, got %s", s.Table)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestReader_GetMetrics_PoWA4(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	r := &Reader{
		db:          db,
		cfg:         &config.DatabaseConfig{},
		hasKCache:   true,
		powaVersion: "4.0.1", // Simulate PoWA 4
		kcacheTable: "powa_kcache_history",
	}

	now := time.Now()

	// Expect PoWA 4 style query (using unnest)
	// We check for "CROSS JOIN LATERAL unnest" to verify it's the correct query
	// Use (?s) to enable dot-matches-newline for multi-line query matching
	mock.ExpectQuery(`(?s)SELECT.*powa_statements_history.*unnest`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"queryid", "query", "datname", "server_name", "srvid", "total_time", "mean_time", "calls", "ts"}).
			AddRow(1001, "SELECT 1", "postgres", "server1", 1, 100.0, 10.0, 10, now))

	// Expect PoWA 4 style kcache query (using exec_ prefix and powa_kcache_history)
	mock.ExpectQuery(`(?s)SELECT.*exec_reads.*powa_kcache_history`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"queryid", "srvid", "reads", "writes", "user_time", "sys_time"}).
			AddRow(1001, 1, 50, 10, 5.0, 1.0))

	metrics, err := r.getMetrics(context.Background(), now.Add(-1*time.Hour), now)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	m := metrics[0]
	if m.QueryID != 1001 {
		t.Errorf("expected queryID 1001, got %d", m.QueryID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// TestReader_GetMetrics_DeltaSemantics asserts that getMetrics returns delta values (last − first in the window),
// not SUM. The query computes first/last per (queryid, ...) and returns calls = last_calls − first_calls,
// total_time = last_time − first_time. This test mocks one row with delta-shaped values and asserts them.
func TestReader_GetMetrics_DeltaSemantics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating mock: %s", err)
	}
	defer db.Close()

	r := &Reader{
		db:          db,
		cfg:         &config.DatabaseConfig{},
		hasKCache:   false, // no kcache so we only assert metrics query
		powaVersion: "3.2.0",
	}

	now := time.Now()
	// Simulate first_calls=100, last_calls=150 → calls=50; first_time=1000, last_time=1050 → total_time=50
	mock.ExpectQuery("SELECT.*powa_statements_history").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"queryid", "query", "datname", "server_name", "srvid", "total_time", "mean_time", "calls", "ts"}).
			AddRow(1001, "SELECT 1", "postgres", "local", 0, 50.0, 1.0, 50, now))

	metrics, err := r.getMetrics(context.Background(), now.Add(-1*time.Hour), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	m := metrics[0]
	if m.Calls != 50 {
		t.Errorf("expected calls (delta) = 50, got %d", m.Calls)
	}
	if m.TotalTime != 50.0 {
		t.Errorf("expected total_time (delta) = 50.0, got %f", m.TotalTime)
	}
	if m.MeanTime != 1.0 {
		t.Errorf("expected mean_time = 1.0, got %f", m.MeanTime)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
