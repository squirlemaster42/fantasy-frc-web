package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryRowValue(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare("SELECT name FROM users WHERE id = \\$1").
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("alice"))

	name, err := QueryRowValue[string](context.Background(), db, "SELECT name FROM users WHERE id = $1", 1)
	require.NoError(t, err)
	assert.Equal(t, "alice", name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryRowValue_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare("SELECT name FROM users WHERE id = \\$1").
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow(nil))

	_, err = QueryRowValue[string](context.Background(), db, "SELECT name FROM users WHERE id = $1", 1)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare("SELECT name FROM users").
		ExpectQuery().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("alice").AddRow("bob"))

	names, err := QueryRows(context.Background(), db, "SELECT name FROM users", func(rows *sql.Rows) (string, error) {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", errors.New("failed to scan row")
		}
		return name, nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob"}, names)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExec(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare("UPDATE users SET name = \\$1 WHERE id = \\$2").
		ExpectExec().
		WithArgs("alice", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := Exec(context.Background(), db, "UPDATE users SET name = $1 WHERE id = $2", "alice", 1)
	require.NoError(t, err)
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
	require.NoError(t, mock.ExpectationsWereMet())
}
