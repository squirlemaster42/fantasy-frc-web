package database

import (
	"context"
	"database/sql"
	"fmt"
)

// QueryRow prepares and executes a single-row query, scans the result using
// scanFn, and returns the value. It is the common backbone for the many
// store functions that previously duplicated this pattern.
func QueryRow[T any](ctx context.Context, db DBTX, query string, scanFn func(*sql.Row) (T, error), args ...any) (T, error) {
	var zero T
	stmt, err := Prepare(ctx, db, query)
	if err != nil {
		return zero, err
	}
	defer CloseStatement(ctx, stmt, "QueryRow")

	result, err := scanFn(stmt.QueryRowContext(ctx, args...))
	if err != nil {
		return zero, err
	}
	return result, nil
}

// QueryRowValue is a convenience wrapper for QueryRow when scanning into a
// single primitive value. The caller provides the destination pointer.
func QueryRowValue[T any](ctx context.Context, db DBTX, query string, args ...any) (T, error) {
	var value T
	return QueryRow(ctx, db, query, func(row *sql.Row) (T, error) {
		if err := row.Scan(&value); err != nil {
			return value, fmt.Errorf("failed to scan row: %w", err)
		}
		return value, nil
	}, args...)
}

// QueryRows prepares and executes a multi-row query, scans each row using
// scanFn, and returns the collected slice.
func QueryRows[T any](ctx context.Context, db DBTX, query string, scanFn func(*sql.Rows) (T, error), args ...any) ([]T, error) {
	stmt, err := Prepare(ctx, db, query)
	if err != nil {
		return nil, err
	}
	defer CloseStatement(ctx, stmt, "QueryRows")

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rows: %w", err)
	}
	defer CloseRows(ctx, rows, "QueryRows")

	results := make([]T, 0)
	for rows.Next() {
		item, err := scanFn(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}
	return results, nil
}

// Exec prepares and executes a query that does not return rows.
func Exec(ctx context.Context, db DBTX, query string, args ...any) (sql.Result, error) {
	stmt, err := Prepare(ctx, db, query)
	if err != nil {
		return nil, err
	}
	defer CloseStatement(ctx, stmt, "Exec")

	result, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute statement: %w", err)
	}
	return result, nil
}
