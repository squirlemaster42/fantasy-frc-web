package metrics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"server/utils"
)

func getDB(t *testing.T) *sql.DB {
	t.Helper()
	envPath := filepath.Join("../", ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: no .env file at %s", envPath)
	}
	if err := utils.LoadEnvFile(envPath); err != nil {
		t.Skipf("Skipping test: failed to load .env file %v", err)
	}

	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbIp := os.Getenv("DB_IP")
	dbName := os.Getenv("DB_NAME")

	if dbUsername == "" || dbPassword == "" || dbIp == "" || dbName == "" {
		t.Skip("Skipping test: database credentials not found in environment")
	}

	connStr := "postgresql://" + dbUsername + ":" + dbPassword + "@" + dbIp + "/" + dbName + "?sslmode=disable"
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Skipf("Skipping test: failed to open database connection %v", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("Skipping test: failed to ping database %v", err)
	}

	return db
}

func TestDBStatsCollector(t *testing.T) {
	db := getDB(t)
	defer func() { _ = db.Close() }()

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewDBStatsCollector(db, "postgres"))

	_, err := db.ExecContext(context.Background(), "SELECT 1")
	require.NoError(t, err)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(rec, req)

	output := rec.Body.String()

	assert.Contains(t, output, "go_sql_in_use_connections",
		"go_sql_in_use_connections metric should be present")
	assert.Contains(t, output, "go_sql_idle_connections",
		"go_sql_idle_connections metric should be present")
	assert.Contains(t, output, "go_sql_max_open_connections",
		"go_sql_max_open_connections metric should be present")

	t.Log("DBStatsCollector metrics are being collected correctly")
}

func TestDBStatsCollectorQueryCount(t *testing.T) {
	db := getDB(t)
	defer func() { _ = db.Close() }()

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewDBStatsCollector(db, "postgres"))

	for range 3 {
		_, err := db.ExecContext(context.Background(), "SELECT 1")
		require.NoError(t, err)
	}

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(rec, req)

	output := rec.Body.String()
	fmt.Printf("Metrics output sample:\n%s\n", strings.Split(output, "\n")[0:10])

	assert.Contains(t, output, "go_sql_in_use_connections",
		"DB connection metrics should be trackable")
}

func TestOTelDBStatsMetrics(t *testing.T) {
	db := getDB(t)
	defer func() { _ = db.Close() }()

	reg, err := otelsql.RegisterDBStatsMetrics(db)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, reg.Unregister())
	}()

	// Trigger some DB activity so stats are non-trivial
	_, err = db.ExecContext(context.Background(), "SELECT 1")
	require.NoError(t, err)

	t.Log("OTel DB stats metrics registered successfully")
}

func TestShutdownMetricsIdempotent(t *testing.T) {
	// Should not panic even when called before InitMetrics
	assert.NotPanics(t, func() {
		ShutdownMetrics()
	}, "ShutdownMetrics should be idempotent and not panic when called before InitMetrics")
}

func TestRecordUserActivity(t *testing.T) {
	activeUsersMu.Lock()
	activeUsers = make(map[string]time.Time)
	activeUsersMu.Unlock()

	RecordUserActivity("user-1")

	activeUsersMu.RLock()
	_, ok := activeUsers["user-1"]
	activeUsersMu.RUnlock()

	assert.True(t, ok, "user should be recorded in active users map")
}

func TestRecordAuthenticatedRequest(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(authenticatedRequestCount)

	RecordAuthenticatedRequest("GET", "/u/home")
	RecordAuthenticatedRequest("POST", "/u/createDraft")

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(rec, req)

	output := rec.Body.String()
	assert.Contains(t, output, "authenticated_requests_total",
		"authenticated_requests_total metric should be present")
	assert.Contains(t, output, `method="GET"`,
		"GET method label should be present")
	assert.Contains(t, output, `route="/u/home"`,
		"/u/home route label should be present")
}

func TestWebSocketListenerGauge(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(websocketListenerGauge)

	IncrementWebSocketListener()
	IncrementWebSocketListener()
	DecrementWebSocketListener()

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(rec, req)

	output := rec.Body.String()
	assert.Contains(t, output, "websocket_listeners_active",
		"websocket_listeners_active metric should be present")
	assert.Contains(t, output, " 1",
		"gauge value should be 1 after 2 increments and 1 decrement")
}

func TestRecordTbaRequest(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(tbaRequestCount)
	registry.MustRegister(tbaRequestDuration)

	RecordTbaRequest("/event/2024cur/teams", http.StatusOK, 150*time.Millisecond)
	RecordTbaRequest("", http.StatusNotFound, 50*time.Millisecond)

	count, err := testutil.GatherAndCount(registry, "tba_requests_total")
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	assert.InDelta(t, 1.0, testutil.ToFloat64(tbaRequestCount.WithLabelValues("/event/2024cur/teams", "200")), 0.0)
	assert.InDelta(t, 1.0, testutil.ToFloat64(tbaRequestCount.WithLabelValues(unknownRouteLabel, "404")), 0.0)
}

func TestRecordTbaCacheHit(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(tbaCacheHits)

	RecordTbaCacheHit("hit")
	RecordTbaCacheHit("miss")
	RecordTbaCacheHit("hit")

	assert.InDelta(t, 2.0, testutil.ToFloat64(tbaCacheHits.WithLabelValues("hit")), 0.0)
	assert.InDelta(t, 1.0, testutil.ToFloat64(tbaCacheHits.WithLabelValues("miss")), 0.0)
}

func TestCollectQueryStatsIteration(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"query", "calls", "mean_exec_time", "total_exec_time", "rows"}).
		AddRow("SELECT very_long_query_text_"+strings.Repeat("x", 200), 10, 0.123, 1.23, 100)

	mock.ExpectQuery("SELECT query, calls, mean_exec_time::numeric, total_exec_time::numeric, rows FROM pg_stat_statements").
		WithArgs(float64(DBQueryThresholdMs())/1000.0, DBQueryMaxCount()).
		WillReturnRows(rows)

	collectQueryStatsIteration(t.Context(), db)

	queryText := "SELECT very_long_query_text_" + strings.Repeat("x", 200)
	queryID := queryText
	if len(queryID) > MetricsQueryIdMaxLength() {
		queryID = queryID[:MetricsQueryIdMaxLength()] + "..."
	}
	assert.InDelta(t, 0.000123, testutil.ToFloat64(dbQueryMeanTime.WithLabelValues(queryID)), 1e-9)
	assert.InDelta(t, 10.0, testutil.ToFloat64(dbQueryCalls.WithLabelValues(queryID)), 0.0)
	assert.InDelta(t, 100.0, testutil.ToFloat64(dbQueryRows.WithLabelValues(queryID)), 0.0)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectQueryStatsIteration_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT query, calls, mean_exec_time::numeric, total_exec_time::numeric, rows FROM pg_stat_statements").
		WithArgs(float64(DBQueryThresholdMs())/1000.0, DBQueryMaxCount()).
		WillReturnError(errors.New("connection refused"))

	// Should not panic and should not touch gauges.
	collectQueryStatsIteration(t.Context(), db)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDefaultMetricsValues(t *testing.T) {
	assert.Equal(t, 10*time.Second, ActiveUserCollectorTickInterval())
	assert.Equal(t, 100, MetricsQueryIdMaxLength())
	assert.Equal(t, 50, DBQueryThresholdMs())
	assert.Equal(t, 30*time.Second, DBQueryPollInterval())
	assert.Equal(t, 50, DBQueryMaxCount())
}

func TestInitMetrics(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	require.NoError(t, InitMetrics(t.Context(), db))

	// Verify HTTP counters can still be incremented after initialization.
	before := testutil.ToFloat64(authenticatedRequestCount.WithLabelValues("GET", "/u/home"))
	RecordAuthenticatedRequest("GET", "/u/home")
	after := testutil.ToFloat64(authenticatedRequestCount.WithLabelValues("GET", "/u/home"))
	assert.InDelta(t, before+1, after, 0.0)

	// Shutdown should not panic after initialization.
	assert.NotPanics(t, func() {
		ShutdownMetrics()
	})
}
