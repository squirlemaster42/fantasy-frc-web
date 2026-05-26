package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/XSAM/otelsql"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestCreateConnectionString(t *testing.T) {
	connStr := createConnectionString("user", "pass", "localhost", "mydb")
	assert.Equal(t, "postgresql://user:pass@localhost/mydb?sslmode=disable", connStr)
}

func TestRegisterDatabaseConnection(t *testing.T) {
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

	// Set up in-memory tracer to capture spans
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	ctx := context.Background()
	db, err := RegisterDatabaseConnection(ctx, dbUsername, dbPassword, dbIp, dbName,
		otelsql.WithTracerProvider(tp),
	)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Execute a query to generate spans
	rows, err := db.QueryContext(ctx, "SELECT 1")
	require.NoError(t, err)
	require.NoError(t, rows.Close())

	// Force flush before checking spans
	err = tp.ForceFlush(ctx)
	require.NoError(t, err)

	spans := exporter.GetSpans()
	for _, span := range spans {
		t.Logf("Captured span: %s", span.Name)
	}
	require.NotEmpty(t, spans, "expected at least one span to be captured")

	// Verify we have SQL spans
	var hasConnect, hasQuery, hasRows bool
	for _, span := range spans {
		switch span.Name {
		case "sql.connector.connect":
			hasConnect = true
		case "sql.conn.query":
			hasQuery = true
		case "sql.rows":
			hasRows = true
		}
	}
	assert.True(t, hasConnect, "expected a sql.connector.connect span")
	assert.True(t, hasQuery, "expected a sql.conn.query span")
	assert.True(t, hasRows, "expected a sql.rows span")

	_ = tp.Shutdown(ctx)
}

func TestRegisterDatabaseConnectionInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	db, err := RegisterDatabaseConnection(ctx, "invalid", "invalid", "127.0.0.1", "invalid")
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to ping database")
}
