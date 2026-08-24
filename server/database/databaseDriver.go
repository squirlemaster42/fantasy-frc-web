package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/assert"
	"server/log"
	"strings"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgconn"
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

func createConnectionString(username string, password string, ip string, dbName string) string {
	return "postgresql://" + username + ":" + password + "@" + ip + "/" + dbName + "?sslmode=disable&timezone=UTC"
}

// Placeholders returns a slice of SQL positional placeholders starting at
// startPos (e.g. startPos=1, count=3 returns ["$1", "$2", "$3"]). The caller
// is responsible for passing the actual values as query arguments; this
// function never mixes user input into the returned strings.
func Placeholders(startPos int, count int) []string {
	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		placeholders[i] = fmt.Sprintf("$%d", startPos+i)
	}
	return placeholders
}

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// isProgrammingError reports whether err is a PostgreSQL schema, syntax, or
// statement error that indicates a code/schema mismatch. These errors should
// crash the process because retrying will not resolve them.
func isProgrammingError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
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
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		if isProgrammingError(err) {
			a := assert.CreateAssertWithContext("Prepare")
			a.AddContext("query", query)
			a.AddContext("sqlstate", sqlState(err))
			a.NoError(ctx, err, "failed to prepare statement due to schema/syntax error")
		}
		log.Error(ctx, "Failed to prepare statement", "error", err, "query", query)
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	return stmt, nil
}

func CloseStatement(ctx context.Context, stmt *sql.Stmt, funcName string) {
	if stmt == nil {
		return
	}
	if err := stmt.Close(); err != nil {
		log.Error(ctx, funcName+": failed to close statement", "error", err)
	}
}

func CloseRows(ctx context.Context, rows *sql.Rows, funcName string) {
	if rows == nil {
		return
	}
	if err := rows.Close(); err != nil {
		log.Error(ctx, funcName+": failed to close rows", "error", err)
	}
}
