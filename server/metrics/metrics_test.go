package metrics

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
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
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test: failed to open database connection %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: failed to ping database %v", err)
	}

	return db
}

func TestDBStatsCollector(t *testing.T) {
	db := getDB(t)
	defer db.Close()

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewDBStatsCollector(db, "postgres"))

	_, err := db.Exec("SELECT 1")
	assert.NoError(t, err)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(rec, req)

	output := rec.Body.String()

	assert.True(t, strings.Contains(output, "go_sql_in_use_connections"),
		"go_sql_in_use_connections metric should be present")
	assert.True(t, strings.Contains(output, "go_sql_idle_connections"),
		"go_sql_idle_connections metric should be present")
	assert.True(t, strings.Contains(output, "go_sql_max_open_connections"),
		"go_sql_max_open_connections metric should be present")

	t.Log("DBStatsCollector metrics are being collected correctly")
}

func TestDBStatsCollectorQueryCount(t *testing.T) {
	db := getDB(t)
	defer db.Close()

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewDBStatsCollector(db, "postgres"))

	for i := 0; i < 3; i++ {
		_, err := db.Exec("SELECT 1")
		assert.NoError(t, err)
	}

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(rec, req)

	output := rec.Body.String()
	fmt.Printf("Metrics output sample:\n%s\n", strings.Split(output, "\n")[0:10])

	assert.True(t, strings.Contains(output, "go_sql_in_use_connections"),
		"DB connection metrics should be trackable")
}
