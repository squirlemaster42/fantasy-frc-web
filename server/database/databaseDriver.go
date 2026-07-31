package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/assert"
	"server/log"
	"strings"
	"time"

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

	db.SetMaxOpenConns(90)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

type DBTX interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

func createConnectionString(username string, password string, ip string, dbName string) string {
	return "postgresql://" + username + ":" + password + "@" + ip + "/" + dbName + "?sslmode=disable&timezone=UTC"
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
		case strings.HasPrefix(pgErr.Code, "42"): // Syntax / Access Rule Violation
			return true
		case strings.HasPrefix(pgErr.Code, "22"): // Data Exception
			return true
		case strings.HasPrefix(pgErr.Code, "26"): // Invalid SQL Statement Name
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
