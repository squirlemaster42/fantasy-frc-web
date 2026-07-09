package background

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestNewCleanupService(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewCleanupService(db, 60)

	assert.NotNil(t, service)
	assert.False(t, service.running)
	assert.Equal(t, 60, service.interval)
}

func TestCleanupService_StartStop(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewCleanupService(db, 60)

	ctx := context.Background()

	err = service.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, service.running)

	err = service.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	err = service.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, service.running)

	err = service.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already stopped")

	// Ensure no unexpected database interactions occurred
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCleanupService_cleanExpiredSessionTokens(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare("Delete from UserSessions").
		WillBeClosed().
		ExpectExec().
		WillReturnResult(sqlmock.NewResult(0, 3))

	service := NewCleanupService(db, 60)

	service.cleanExpiredSessionTokens(context.Background())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCleanupService_cleanExpiredSessionTokens_PrepareError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare("Delete from UserSessions").
		WillReturnError(assert.AnError)

	service := NewCleanupService(db, 60)

	// Should not panic on prepare error
	service.cleanExpiredSessionTokens(context.Background())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCleanupService_cleanExpiredSessionTokens_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare("Delete from UserSessions").
		WillBeClosed().
		ExpectExec().
		WillReturnError(assert.AnError)

	service := NewCleanupService(db, 60)

	// Should not panic on exec error
	service.cleanExpiredSessionTokens(context.Background())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCleanupService_Start_RunsCleanup(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare("Delete from UserSessions").
		WillBeClosed().
		ExpectExec().
		WillReturnResult(sqlmock.NewResult(0, 1))

	service := NewCleanupService(db, 1)

	ctx := context.Background()
	err = service.Start(ctx)
	assert.NoError(t, err)

	// Wait for at least one cleanup iteration
	time.Sleep(100 * time.Millisecond)

	err = service.Stop(ctx)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
