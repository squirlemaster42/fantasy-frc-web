package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/log"
	"time"

	"github.com/XSAM/otelsql"
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

	db.SetMaxOpenConns(90)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

func createConnectionString(username string, password string, ip string, dbName string) string {
	return "postgresql://" + username + ":" + password + "@" + ip + "/" + dbName + "?sslmode=disable&timezone=UTC"
}

func Prepare(ctx context.Context, db *sql.DB, query string) (*sql.Stmt, error) {
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		log.Fatal(ctx, "Failed to prepare statement", "error", err, "query", query)
	}
	return stmt, nil
}
