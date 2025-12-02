package model

import (
	"crypto"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetUserBySessionToken_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		sessionToken  string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name:         "database connection error",
			sessionToken: "test-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// No expectations
			},
			expectedError: "Failed to prepare statement",
		},
		{
			name:         "session not found",
			sessionToken: "invalid-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				hasher := crypto.SHA256.New()
				hasher.Write([]byte("invalid-token"))
				hashedToken := hasher.Sum(nil)

				mock.ExpectPrepare(`Select UserUuid From UserSessions Where sessionToken = \$1 and now\(\)::timestamp <= expirationTime;`).
					ExpectQuery().
					WithArgs(hashedToken).
					WillReturnRows(sqlmock.NewRows([]string{"UserUuid"}))
			},
			expectedError: "Failed to get user",
		},
		{
			name:         "session expired",
			sessionToken: "expired-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				hasher := crypto.SHA256.New()
				hasher.Write([]byte("expired-token"))
				hashedToken := hasher.Sum(nil)

				mock.ExpectPrepare(`Select UserUuid From UserSessions Where sessionToken = \$1 and now\(\)::timestamp <= expirationTime;`).
					ExpectQuery().
					WithArgs(hashedToken).
					WillReturnRows(sqlmock.NewRows([]string{"UserUuid"}))
			},
			expectedError: "Failed to get user",
		},
		{
			name:         "username query error",
			sessionToken: "valid-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				hasher := crypto.SHA256.New()
				hasher.Write([]byte("valid-token"))
				hashedToken := hasher.Sum(nil)

				mock.ExpectPrepare(`Select UserUuid From UserSessions Where sessionToken = \$1 and now\(\)::timestamp <= expirationTime;`).
					ExpectQuery().
					WithArgs(hashedToken).
					WillReturnRows(sqlmock.NewRows([]string{"UserUuid"}).AddRow("550e8400-e29b-41d4-a716-446655440000"))

				mock.ExpectPrepare(`Update UserSessions Set expirationTime = now\(\)::timestamp \+ '10 days' Where userUuid = \$1 And sessionToken = \$2;`).
					ExpectExec().
					WithArgs("550e8400-e29b-41d4-a716-446655440000", hashedToken).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectPrepare(`Select Username From Users Where UserUuid = \$1;`).
					ExpectQuery().
					WithArgs("550e8400-e29b-41d4-a716-446655440000").
					WillReturnError(errors.New("username query failed"))
			},
			expectedError: "Failed to prepare statement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			result := GetUserBySessionToken(db, tt.sessionToken)

			assert.Equal(t, uuid.Nil, result) // Error cases return zero UUID

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestValidateLogin_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		password      string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name:     "database connection error",
			username: "testuser",
			password: "password",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// No expectations
			},
			expectedError: "Failed to prepare statement",
		},
		{
			name:     "user not found",
			username: "nonexistent",
			password: "password",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT (.+) FROM Users WHERE Username = \$1`).
					ExpectQuery().
					WithArgs("nonexistent").
					WillReturnRows(sqlmock.NewRows([]string{"UserUuid", "Username", "Password"}))
			},
			expectedError: "Failed to validate login",
		},
		{
			name:     "password scan error",
			username: "testuser",
			password: "password",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT (.+) FROM Users WHERE Username = \$1`).
					ExpectQuery().
					WithArgs("testuser").
					WillReturnRows(sqlmock.NewRows([]string{"UserUuid", "Username", "Password"}).
						AddRow("550e8400-e29b-41d4-a716-446655440000", "testuser", "not-a-hash"))
			},
			expectedError: "Failed to validate login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			result := ValidateLogin(db, tt.username, tt.password)

			assert.False(t, result) // Error cases return false

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRegisterUser_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		password      string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name:     "database connection error",
			username: "newuser",
			password: "password123",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// No expectations
			},
			expectedError: "Failed to prepare statement",
		},
		{
			name:     "insert execution error",
			username: "newuser",
			password: "password123",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`INSERT INTO Users (.+) VALUES (.+) Returning UserUuid;`).
					ExpectQuery().
					WithArgs("newuser", sqlmock.AnyArg()).
					WillReturnError(errors.New("insert failed"))
			},
			expectedError: "Failed to register user",
		},
		{
			name:     "scan error",
			username: "newuser",
			password: "password123",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`INSERT INTO Users (.+) VALUES (.+) Returning UserUuid;`).
					ExpectQuery().
					WithArgs("newuser", sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"UserUuid"}).AddRow("not-a-uuid"))
			},
			expectedError: "Failed to register user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			result := RegisterUser(db, tt.username, tt.password)

			assert.Equal(t, uuid.Nil, result) // Error cases return zero UUID

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUsernameTaken_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name:     "database connection error",
			username: "testuser",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// No expectations
			},
			expectedError: "Failed to prepare statement",
		},
		{
			name:     "query execution error",
			username: "testuser",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT COUNT\(\*\) FROM Users WHERE Username = \$1`).
					ExpectQuery().
					WithArgs("testuser").
					WillReturnError(errors.New("query failed"))
			},
			expectedError: "Failed to check username",
		},
		{
			name:     "scan error",
			username: "testuser",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT COUNT\(\*\) FROM Users WHERE Username = \$1`).
					ExpectQuery().
					WithArgs("testuser").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow("not-a-number"))
			},
			expectedError: "Failed to check username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			result, err := UsernameTaken(db, tt.username)

			assert.Error(t, err)
			assert.False(t, result) // Error cases return false

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRegisterSession_ErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		userUUID      string
		sessionToken  string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError string
	}{
		{
			name:         "database connection error",
			userUUID:     "550e8400-e29b-41d4-a716-446655440000",
			sessionToken: "session-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// No expectations
			},
			expectedError: "Failed to prepare statement",
		},
		{
			name:         "insert execution error",
			userUUID:     "550e8400-e29b-41d4-a716-446655440000",
			sessionToken: "session-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				hasher := crypto.SHA256.New()
				hasher.Write([]byte("session-token"))
				hashedToken := hasher.Sum(nil)

				mock.ExpectPrepare(`INSERT INTO UserSessions (.+) VALUES (.+)`).
					ExpectExec().
					WithArgs("550e8400-e29b-41d4-a716-446655440000", hashedToken, sqlmock.AnyArg()).
					WillReturnError(errors.New("insert failed"))
			},
			expectedError: "Failed to register session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			RegisterSession(db, parseUUID(tt.userUUID), tt.sessionToken)

			// RegisterSession doesn't return error, but we can check expectations
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Helper function
func parseUUID(s string) uuid.UUID {
	u, _ := uuid.Parse(s)
	return u
}
