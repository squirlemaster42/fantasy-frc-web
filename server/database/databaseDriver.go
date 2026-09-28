package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/assert"
	"server/log"
	"strings"
	"sync"

	"github.com/XSAM/otelsql"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func RegisterDatabaseConnection(ctx context.Context, username string, password string, ip string, dbName string, opts ...otelsql.Option) (*sql.DB, error) {
	log.Info(ctx, "Setting up DB connection", "username", username, "ip", ip, "databaseName", dbName)
	connStr := createConnectionString(username, password, ip, dbName)

	attrs := append(
		otelsql.AttributesFromDSN(connStr),
		semconv.DBSystemPostgreSQL,
	)

	options := append([]otelsql.Option{
		otelsql.WithAttributes(attrs...),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			OmitConnResetSession: true,
		}),
	}, opts...)

	driverName, err := otelsql.Register("pgx", options...)
	if err != nil {
		return nil, fmt.Errorf("could not register otelsql driver: %w", err)
	}

	db, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("could not open database connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(MaxOpenConns())
	db.SetMaxIdleConns(MaxIdleConns())
	db.SetConnMaxLifetime(ConnMaxLifetime())

	return db, nil
}

type DBTX interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

var (
	// dbStmtCache maps *sql.DB -> query string -> *sql.Stmt.
	// It stores prepared statements that are safe to reuse across calls.
	dbStmtCache sync.Map // map[*sql.DB]*sync.Map

	// cachedStmts tracks every statement currently held in dbStmtCache.
	// It allows CloseStatement to skip closing cached statements in O(1).
	cachedStmts sync.Map // map[*sql.Stmt]struct{}
)

// getOrCreateDBCache returns the per-DB statement cache for sqlDB.
func getOrCreateDBCache(sqlDB *sql.DB) *sync.Map {
	if cache, ok := dbStmtCache.Load(sqlDB); ok {
		return cache.(*sync.Map)
	}
	newCache := &sync.Map{}
	if cache, loaded := dbStmtCache.LoadOrStore(sqlDB, newCache); loaded {
		return cache.(*sync.Map)
	}
	return newCache
}

// isCachedStatement reports whether stmt is currently stored in dbStmtCache.
func isCachedStatement(stmt *sql.Stmt) bool {
	_, ok := cachedStmts.Load(stmt)
	return ok
}

func createConnectionString(username string, password string, ip string, dbName string) string {
	return "postgresql://" + username + ":" + password + "@" + ip + "/" + dbName + "?sslmode=disable&timezone=UTC"
}

// Placeholders returns a slice of SQL positional placeholders starting at
// startPos (e.g. startPos=1, count=3 returns ["$1", "$2", "$3"]). The caller
// is responsible for passing the actual values as query arguments; this
// function never mixes user input into the returned strings.
func Placeholders(startPos int, count int) []string {
	placeholders := make([]string, count)
	for i := range count {
		placeholders[i] = fmt.Sprintf("$%d", startPos+i)
	}
	return placeholders
}

func sqlState(err error) string {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code
	}
	return ""
}

// isProgrammingError reports whether err is a PostgreSQL schema, syntax, or
// statement error that indicates a code/schema mismatch. These errors should
// crash the process because retrying will not resolve them.
func isProgrammingError(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch {
		case strings.HasPrefix(pgErr.Code, sqlStateSyntaxPrefix): // Syntax / Access Rule Violation
			return true
		case strings.HasPrefix(pgErr.Code, sqlStateDataExceptionPrefix): // Data Exception
			return true
		case strings.HasPrefix(pgErr.Code, sqlStateInvalidStatementPrefix): // Invalid SQL Statement Name
			return true
		}
	}
	return false
}

func Prepare(ctx context.Context, db DBTX, query string) (*sql.Stmt, error) {
	// Only *sql.DB statements can be safely cached. *sql.Tx statements are
	// valid only for the lifetime of their transaction.
	if sqlDB, ok := db.(*sql.DB); ok && sqlDB != nil {
		cache := getOrCreateDBCache(sqlDB)
		if stmt, ok := cache.Load(query); ok {
			return stmt.(*sql.Stmt), nil
		}

		stmt, err := db.PrepareContext(ctx, query)
		if err != nil {
			return nil, handlePrepareError(ctx, query, err)
		}

		if existing, loaded := cache.LoadOrStore(query, stmt); loaded {
			// Another goroutine prepared and stored first; use theirs.
			if closeErr := stmt.Close(); closeErr != nil {
				log.Error(ctx, "Prepare: failed to close duplicate prepared statement", "error", closeErr, "query", query)
			}
			return existing.(*sql.Stmt), nil
		}

		cachedStmts.Store(stmt, struct{}{})
		return stmt, nil
	}

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, handlePrepareError(ctx, query, err)
	}
	return stmt, nil
}

// handlePrepareError classifies prepare errors and logs/crashes appropriately.
func handlePrepareError(ctx context.Context, query string, err error) error {
	if isProgrammingError(err) {
		a := assert.CreateAssertWithContext("Prepare")
		a.AddContext("query", query)
		a.AddContext("sqlstate", sqlState(err))
		a.NoError(ctx, err, "failed to prepare statement due to schema/syntax error")
	}
	log.Error(ctx, "Failed to prepare statement", "error", err, "query", query)
	return fmt.Errorf("failed to prepare statement: %w", err)
}

func CloseStatement(ctx context.Context, stmt *sql.Stmt, funcName string) {
	if stmt == nil {
		return
	}
	if isCachedStatement(stmt) {
		return
	}
	if err := stmt.Close(); err != nil {
		log.Error(ctx, funcName+": failed to close statement", "error", err)
	}
}

// CloseCachedStatements closes and removes all cached prepared statements for db.
// Call this when a *sql.DB is being closed (e.g., graceful shutdown, test cleanup).
func CloseCachedStatements(ctx context.Context, db *sql.DB) {
	cacheValue, ok := dbStmtCache.Load(db)
	if !ok {
		return
	}
	cache := cacheValue.(*sync.Map)
	cache.Range(func(query, stmtValue interface{}) bool {
		stmt := stmtValue.(*sql.Stmt)
		cachedStmts.Delete(stmt)
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "CloseCachedStatements: failed to close cached statement", "error", err, "query", query)
		}
		return true
	})
	dbStmtCache.Delete(db)
}

func CloseRows(ctx context.Context, rows *sql.Rows, funcName string) {
	if rows == nil {
		return
	}
	if err := rows.Close(); err != nil {
		log.Error(ctx, funcName+": failed to close rows", "error", err)
	}
}
