package database

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/XSAM/otelsql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestCreateConnectionString(t *testing.T) {
	connStr := createConnectionString("user", "pass", "localhost", "mydb")
	assert.Equal(t, "postgresql://user:pass@localhost/mydb?sslmode=disable&timezone=UTC", connStr)
}

func TestPlaceholders(t *testing.T) {
	assert.Empty(t, Placeholders(1, 0))
	assert.Equal(t, []string{"$1"}, Placeholders(1, 1))
	assert.Equal(t, []string{"$1", "$2", "$3"}, Placeholders(1, 3))
	assert.Equal(t, []string{"$3", "$4", "$5"}, Placeholders(3, 3))
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
	defer func() { _ = db.Close() }()

	// Execute a query to generate spans
	rows, err := db.QueryContext(ctx, "SELECT 1")
	require.NoError(t, err)
	require.NoError(t, rows.Err())
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
	require.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to ping database")
}

type failingDBTX struct {
	err error
}

func (f *failingDBTX) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, f.err
}

func TestIsProgrammingError(t *testing.T) {
	assert.True(t, isProgrammingError(&pgconn.PgError{Code: "42601"}), "syntax error should be a programming error")
	assert.True(t, isProgrammingError(&pgconn.PgError{Code: "42P01"}), "undefined table should be a programming error")
	assert.True(t, isProgrammingError(&pgconn.PgError{Code: "42703"}), "undefined column should be a programming error")
	assert.True(t, isProgrammingError(&pgconn.PgError{Code: "22003"}), "data exception should be a programming error")
	assert.True(t, isProgrammingError(&pgconn.PgError{Code: "26000"}), "invalid sql statement name should be a programming error")

	assert.False(t, isProgrammingError(&pgconn.PgError{Code: "08006"}), "connection failure should not be a programming error")
	assert.False(t, isProgrammingError(&pgconn.PgError{Code: "53300"}), "too many connections should not be a programming error")
	assert.False(t, isProgrammingError(errors.New("random error")), "non-pg error should not be a programming error")
}

func TestPrepare_ReturnsTransientError(t *testing.T) {
	transientErr := errors.New("transient failure")
	stmt, err := Prepare(context.Background(), &failingDBTX{err: transientErr}, "SELECT 1")
	require.Error(t, err)
	assert.Nil(t, stmt)
	require.ErrorIs(t, err, transientErr)
}

func TestPrepare_ReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stmt, err := Prepare(ctx, &failingDBTX{err: context.Canceled}, "SELECT 1")
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, stmt)
}

func TestPrepare_CachesStatementsForSameDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		CloseCachedStatements(context.Background(), db)
		_ = db.Close()
	}()

	query := "SELECT 1"
	mock.ExpectPrepare(query)

	stmt1, err := Prepare(context.Background(), db, query)
	require.NoError(t, err)
	require.NotNil(t, stmt1)

	stmt2, err := Prepare(context.Background(), db, query)
	require.NoError(t, err)
	require.NotNil(t, stmt2)

	assert.Equal(t, stmt1, stmt2, "expected cached statement to be reused")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrepare_DifferentDBsDoNotShareCache(t *testing.T) {
	db1, mock1, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		CloseCachedStatements(context.Background(), db1)
		_ = db1.Close()
	}()

	db2, mock2, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		CloseCachedStatements(context.Background(), db2)
		_ = db2.Close()
	}()

	query := "SELECT 1"
	mock1.ExpectPrepare(query)
	mock2.ExpectPrepare(query)

	stmt1, err := Prepare(context.Background(), db1, query)
	require.NoError(t, err)

	stmt2, err := Prepare(context.Background(), db2, query)
	require.NoError(t, err)

	assert.NotEqual(t, stmt1, stmt2, "expected different DBs to have different cached statements")
	require.NoError(t, mock1.ExpectationsWereMet())
	require.NoError(t, mock2.ExpectationsWereMet())
}

func TestPrepare_TransactionStatementsAreNotCached(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		CloseCachedStatements(context.Background(), db)
		_ = db.Close()
	}()

	query := "SELECT 1"
	mock.ExpectBegin()
	mock.ExpectPrepare(query)
	mock.ExpectRollback()

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	stmt, err := Prepare(context.Background(), tx, query)
	require.NoError(t, err)
	require.NotNil(t, stmt)
	assert.False(t, isCachedStatement(stmt), "transaction statements should not be cached")

	CloseStatement(context.Background(), stmt, "TestPrepare_TransactionStatementsAreNotCached")
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCloseStatement_DoesNotCloseCachedStatement(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		CloseCachedStatements(context.Background(), db)
		_ = db.Close()
	}()

	query := "SELECT 1"
	mock.ExpectPrepare(query)

	stmt, err := Prepare(context.Background(), db, query)
	require.NoError(t, err)
	require.True(t, isCachedStatement(stmt))

	// CloseStatement should be a no-op for cached statements.
	CloseStatement(context.Background(), stmt, "TestCloseStatement_DoesNotCloseCachedStatement")

	// A subsequent prepare should return the exact same statement.
	stmt2, err := Prepare(context.Background(), db, query)
	require.NoError(t, err)
	assert.Equal(t, stmt, stmt2)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCloseCachedStatements_ClosesAndRemovesCache(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	query := "SELECT 1"
	mock.ExpectPrepare(query)

	stmt, err := Prepare(context.Background(), db, query)
	require.NoError(t, err)
	require.True(t, isCachedStatement(stmt))

	CloseCachedStatements(context.Background(), db)

	assert.False(t, isCachedStatement(stmt), "statement should no longer be tracked after cleanup")

	// After cleanup, preparing the same query should create a new statement.
	mock.ExpectPrepare(query)
	stmt2, err := Prepare(context.Background(), db, query)
	require.NoError(t, err)
	assert.NotEqual(t, stmt, stmt2)
	require.NoError(t, mock.ExpectationsWereMet())
}
