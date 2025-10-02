package model

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetDraft_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		draftID       int
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name:    "database connection error",
			draftID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`Select
        DisplayName,
        COALESCE\(Description, ''\) As Description,
        COALESCE\(Status, ''\) As Status,
        StartTime,
        EndTime,
        extract\('epoch' from Interval\)::int As Interval,
        OwnerUserUuid
    From Drafts Where Id = \$1;`).
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: "sql: connection is already closed",
		},
		{
			name:    "draft not found",
			draftID: 999,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`Select
        DisplayName,
        COALESCE\(Description, ''\) As Description,
        COALESCE\(Status, ''\) As Status,
        StartTime,
        EndTime,
        extract\('epoch' from Interval\)::int As Interval,
        OwnerUserUuid
    From Drafts Where Id = \$1;`).
					ExpectQuery().
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: "failed to load draft",
		},
		{
			name:    "database scan error",
			draftID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`Select
        DisplayName,
        COALESCE\(Description, ''\) As Description,
        COALESCE\(Status, ''\) As Status,
        StartTime,
        EndTime,
        extract\('epoch' from Interval\)::int As Interval,
        OwnerUserUuid
    From Drafts Where Id = \$1;`).
					ExpectQuery().
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{
						"DisplayName", "Description", "Status", "StartTime", "EndTime", "Interval", "OwnerUserUuid",
					}).AddRow("Test", "Desc", "Filling", "2024-01-01", "2024-01-02", "not-a-number", "uuid"))
			},
			expectedError: "failed to load draft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			_, err = GetDraft(db, tt.draftID)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCreateDraft_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		draft         *DraftModel
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name: "database connection error",
			draft: &DraftModel{
				DisplayName: "Test Draft",
				Description: "Test Description",
				Interval:    30,
				StartTime:   parseTime("2024-01-01T10:00:00Z"),
				EndTime:     parseTime("2024-01-02T10:00:00Z"),
				Owner:       User{UserUuid: parseUUID("550e8400-e29b-41d4-a716-446655440000")},
				Status:      "Filling",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Don't set up expectations
			},
			expectedError: "Failed to prepare statement",
		},
		{
			name: "insert execution error",
			draft: &DraftModel{
				DisplayName: "Test Draft",
				Description: "Test Description",
				Interval:    30,
				StartTime:   parseTime("2024-01-01T10:00:00Z"),
				EndTime:     parseTime("2024-01-02T10:00:00Z"),
				Owner:       User{UserUuid: parseUUID("550e8400-e29b-41d4-a716-446655440000")},
				Status:      "Filling",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`INSERT INTO Drafts (.+)`).
					ExpectQuery().
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errors.New("insert failed"))
			},
			expectedError: "Failed to create draft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			result, err := CreateDraft(db, tt.draft)

			assert.Error(t, err)
			assert.Equal(t, -1, result) // Error case returns -1

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateDraftStatus_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		draftID       int
		status        DraftState
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name:    "database connection error",
			draftID: 1,
			status:  "Picking",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// No expectations
			},
			expectedError: "Failed to prepare statement",
		},
		{
			name:    "update execution error",
			draftID: 1,
			status:  "Picking",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`Update Drafts Set Status = \$1 Where Id = \$2;`).
					ExpectExec().
					WithArgs("Picking", 1).
					WillReturnError(errors.New("update failed"))
			},
			expectedError: "Failed to update draft status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			err = UpdateDraftStatus(db, tt.draftID, tt.status)

			assert.Error(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetDraftsForUser_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		userUUID      string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name:     "database connection error",
			userUUID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT DISTINCT.*`).
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: "sql: connection is already closed",
		},
		{
			name:     "query execution error",
			userUUID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT DISTINCT.*`).
					ExpectQuery().
					WithArgs("Filling", "550e8400-e29b-41d4-a716-446655440000").
					WillReturnError(errors.New("query failed"))
			},
			expectedError: "query failed",
		},
		{
			name:     "scan error",
			userUUID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT DISTINCT.*`).
					ExpectQuery().
					WithArgs("Filling", "550e8400-e29b-41d4-a716-446655440000").
					WillReturnRows(sqlmock.NewRows([]string{"Id", "displayName", "ownerId", "OwnerUsername", "Status"}).AddRow("not-a-number", "Test", "uuid", "user", "Filling"))
			},
			expectedError: "not-a-number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			result, err := GetDraftsForUser(db, parseUUID(tt.userUUID))

			assert.Error(t, err)
			assert.Nil(t, result) // Error cases return nil

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInvitePlayer_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		draftID       int
		invitingUUID  string
		invitedUUID   string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name:         "database connection error",
			draftID:      1,
			invitingUUID: "550e8400-e29b-41d4-a716-446655440000",
			invitedUUID:  "550e8400-e29b-41d4-a716-446655440001",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// No expectations
			},
			expectedError: "Failed to prepare statement",
		},
		{
			name:         "insert execution error",
			draftID:      1,
			invitingUUID: "550e8400-e29b-41d4-a716-446655440000",
			invitedUUID:  "550e8400-e29b-41d4-a716-446655440001",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`INSERT INTO DraftInvites.*`).
					ExpectQuery().
					WithArgs(1, "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001", sqlmock.AnyArg(), false).
					WillReturnError(errors.New("insert failed"))
			},
			expectedError: "insert failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			result, err := InvitePlayer(db, tt.draftID, parseUUID(tt.invitingUUID), parseUUID(tt.invitedUUID))

			assert.Error(t, err)
			assert.Equal(t, -1, result) // Error case returns -1

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Helper functions for tests
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
