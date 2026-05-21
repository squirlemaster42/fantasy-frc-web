package metrics

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func getDB(t *testing.T) *sql.DB {
	err := godotenv.Load(filepath.Join("../", ".env"))
	if err != nil {
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

func setupTestMeterProvider() *sdkmetric.ManualReader {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)
	return reader
}

func resetMeterProvider() {
	otel.SetMeterProvider(sdkmetric.NewMeterProvider())
}

func TestInitMetrics(t *testing.T) {
	db := getDB(t)
	defer db.Close()

	setupTestMeterProvider()
	defer resetMeterProvider()

	err := InitMetrics(context.Background(), db)
	assert.NoError(t, err)

	ShutdownMetrics()
}

func TestShutdownMetricsIdempotent(t *testing.T) {
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

func TestMetricsMiddleware(t *testing.T) {
	reader := setupTestMeterProvider()
	defer resetMeterProvider()

	InitHTTPMetrics()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/home", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := MetricsMiddleware()
	handler := mw(func(c echo.Context) error {
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	err := handler(c)
	assert.NoError(t, err)

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "http.request.count" {
				found = true
			}
		}
	}
	assert.True(t, found, "http.request.count metric should be present")
}

func TestRecordAuthenticatedRequest(t *testing.T) {
	reader := setupTestMeterProvider()
	defer resetMeterProvider()

	InitHTTPMetrics()

	RecordAuthenticatedRequest("GET", "/u/home")
	RecordAuthenticatedRequest("POST", "/u/createDraft")

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "http.authenticated.request.count" {
				found = true
			}
		}
	}
	assert.True(t, found, "http.authenticated.request.count metric should be present")
}

func TestWebSocketListenerGauge(t *testing.T) {
	reader := setupTestMeterProvider()
	defer resetMeterProvider()

	InitWebSocketMetrics()

	IncrementWebSocketListener()
	IncrementWebSocketListener()
	DecrementWebSocketListener()

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "websocket.listeners.active" {
				found = true
			}
		}
	}
	assert.True(t, found, "websocket.listeners.active metric should be present")
}

func TestOTelDBStatsMetrics(t *testing.T) {
	db := getDB(t)
	defer db.Close()

	reader := setupTestMeterProvider()
	defer resetMeterProvider()

	err := InitMetrics(context.Background(), db)
	assert.NoError(t, err)

	_, err = db.ExecContext(context.Background(), "SELECT 1")
	assert.NoError(t, err)

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "db.sql.connection.open" {
				found = true
			}
		}
	}
	assert.True(t, found, "OTel DB stats metrics should be present")

	ShutdownMetrics()
}
