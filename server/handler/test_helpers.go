package handler

import (
	"crypto"
	"database/sql"
	"log/slog"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// TestHelper provides common test utilities for handler tests
type TestHelper struct {
	DB   *sql.DB
	Mock sqlmock.Sqlmock
}

// NewTestHelper creates a new test helper with mocked database
func NewTestHelper(t *testing.T) *TestHelper {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	return &TestHelper{
		DB:   db,
		Mock: mock,
	}
}

// Close cleans up the test helper
func (th *TestHelper) Close() {
	err := th.DB.Close()
    if err != nil {
        slog.Warn("Failde to close db connection", "Error", err)
    }
}

// CreateMockHandler creates a handler with mocked dependencies
func (th *TestHelper) CreateMockHandler() *Handler {
	return &Handler{
		Database: th.DB,
		// Other dependencies can be mocked here as needed
	}
}

// MockUserBySessionToken sets up a mock for GetUserBySessionToken
func (th *TestHelper) MockUserBySessionToken(sessionToken string, userUUID string, username string) {
	// Hash the session token as the actual code does
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	hashedToken := hasher.Sum(nil)

	th.Mock.ExpectPrepare(`Select UserUuid From UserSessions Where sessionToken = \$1 and now\(\)::timestamp <= expirationTime;`).
		ExpectQuery().
		WithArgs(hashedToken).
		WillReturnRows(sqlmock.NewRows([]string{"UserUuid"}).AddRow(userUUID))

	// Mock the session expiration update (called immediately after getting user)
	th.Mock.ExpectPrepare(`Update UserSessions Set expirationTime = now\(\)::timestamp \+ '10 days' Where userUuid = \$1 And sessionToken = \$2;`).
		ExpectExec().
		WithArgs(userUUID, hashedToken).
		WillReturnResult(sqlmock.NewResult(1, 1))

	th.Mock.ExpectPrepare(`Select Username From Users Where UserUuid = \$1;`).
		ExpectQuery().
		WithArgs(userUUID).
		WillReturnRows(sqlmock.NewRows([]string{"Username"}).AddRow(username))
}

// MockInvalidSessionToken sets up a mock for an invalid session token
func (th *TestHelper) MockInvalidSessionToken(sessionToken string) {
	// Hash the session token as the actual code does
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	hashedToken := hasher.Sum(nil)

	th.Mock.ExpectPrepare(`Select UserUuid From UserSessions Where sessionToken = \$1 and now\(\)::timestamp <= expirationTime;`).
		ExpectQuery().
		WithArgs(hashedToken).
		WillReturnRows(sqlmock.NewRows([]string{"UserUuid"}))
}

// MockDraftRetrieval sets up a mock for GetDraft
func (th *TestHelper) MockDraftRetrieval(draftID int, exists bool, ownerUUID string) {
	if exists {
		th.Mock.ExpectPrepare(`Select DisplayName, COALESCE\(Description, ''\) As Description, COALESCE\(Status, ''\) As Status, StartTime, EndTime, extract\('epoch' from Interval\)::int As Interval, OwnerUserUuid From Drafts Where Id = \$1;`).
			ExpectQuery().
			WithArgs(draftID).
			WillReturnRows(sqlmock.NewRows([]string{
				"DisplayName", "Description", "Status", "StartTime", "EndTime", "Interval", "OwnerUserUuid",
			}).AddRow("Test Draft", "Test Description", "Filling", "2024-01-01T10:00:00Z", "2024-01-02T10:00:00Z", 30, ownerUUID))
	} else {
		th.Mock.ExpectPrepare(`Select DisplayName, COALESCE\(Description, ''\) As Description, COALESCE\(Status, ''\) As Status, StartTime, EndTime, extract\('epoch' from Interval\)::int As Interval, OwnerUserUuid From Drafts Where Id = \$1;`).
			ExpectQuery().
			WithArgs(draftID).
			WillReturnError(sql.ErrNoRows)
	}
}

// MockDraftUpdate sets up a mock for UpdateDraft
func (th *TestHelper) MockDraftUpdate(draftID int, success bool) {
	if success {
		th.Mock.ExpectExec(`UPDATE Drafts SET (.+) WHERE Id = \$1`).
			WithArgs(draftID).
			WillReturnResult(sqlmock.NewResult(1, 1))
	} else {
		th.Mock.ExpectExec(`UPDATE Drafts SET (.+) WHERE Id = \$1`).
			WithArgs(draftID).
			WillReturnError(sql.ErrNoRows)
	}
}

// MockUserSearch sets up a mock for SearchUsers
func (th *TestHelper) MockUserSearch(searchTerm string, draftID int, results []string) {
	rows := sqlmock.NewRows([]string{"UserUuid", "Username"})
	for _, username := range results {
		rows.AddRow("uuid-"+username, username)
	}

	th.Mock.ExpectQuery(`SELECT (.+) FROM Users WHERE (.+)`).
		WithArgs("%"+searchTerm+"%", draftID).
		WillReturnRows(rows)
}

// MockInviteCreation sets up a mock for InvitePlayer
func (th *TestHelper) MockInviteCreation(draftID int, invitingUUID, invitedUUID string, inviteID int) {
	th.Mock.ExpectQuery(`INSERT INTO DraftInvites (.+) VALUES (.+) RETURNING id`).
		WithArgs(draftID, invitingUUID, invitedUUID, sqlmock.AnyArg(), false).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(inviteID))
}

// AssertExpectations verifies that all mock expectations were met
func (th *TestHelper) AssertExpectations(t *testing.T) {
	assert.NoError(t, th.Mock.ExpectationsWereMet())
}
