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

	// Expect check for pg_stat_kcache
	mock.ExpectQuery("SELECT EXISTS.*pg_stat_kcache").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Expect check for pg_qualstats
	mock.ExpectQuery("SELECT EXISTS.*pg_qualstats").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

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

func TestReader_GetMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	r := &Reader{
		db:        db,
		cfg:       &config.DatabaseConfig{},
		hasKCache: true, // Simulate kcache available
	}

	// Mock data
	now := time.Now()

	// Expect main metrics query
	// Note: We use a regex for the query matching because whitespace/formatting might vary
	mock.ExpectQuery("SELECT.*powa_statements_history").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"queryid", "query", "datname", "total_time", "mean_time", "calls", "ts"}).
			AddRow(1001, "SELECT 1", "postgres", 100.0, 10.0, 10, now))

	// Expect kcache enrichment query (since hasKCache=true)
	mock.ExpectQuery("SELECT.*powa_kcache_metrics_history").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"queryid", "reads", "writes", "user_time", "sys_time"}).
			AddRow(1001, 50, 10, 5.0, 1.0))

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
