package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"server/database"
	"server/utils"
)

var (
	testDB     *sql.DB
	testDBOnce sync.Once
	testDBErr  error
)

// setupTestDB loads environment variables and returns a shared database connection.
// Tests that require the database should call this and skip if credentials are unavailable.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	testDBOnce.Do(func() {
		_ = utils.LoadEnvFile(filepath.Join("..", ".env"))

		dbUsername := os.Getenv("DB_USERNAME")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbIp := os.Getenv("DB_IP")
		dbName := os.Getenv("DB_NAME")

		if dbUsername == "" || dbPassword == "" || dbIp == "" || dbName == "" {
			testDBErr = sql.ErrConnDone // marker to skip tests
			return
		}

		testDB, testDBErr = database.RegisterDatabaseConnection(context.Background(), dbUsername, dbPassword, dbIp, dbName)
		if testDBErr == nil {
			testDBErr = ensureTestTeams(context.Background(), testDB)
		}
	})

	if errors.Is(testDBErr, sql.ErrConnDone) {
		t.Skip("Skipping test: database credentials not found in environment")
	}
	require.NoError(t, testDBErr)
	require.NotNil(t, testDB)

	return testDB
}

// ensureTestTeams seeds the Teams table with team keys referenced by the
// integration tests. It is idempotent so a fresh container or an existing
// shared database both work.
func ensureTestTeams(ctx context.Context, db *sql.DB) error {
	teams := []string{
		"frc254", "frc971", "frcfinal",
		"frcteamA", "frcteamB",
	}
	for _, tbaId := range teams {
		_, err := db.ExecContext(ctx,
			"INSERT INTO Teams (tbaId, name, allianceScore) VALUES ($1, $2, $3) ON CONFLICT (tbaId) DO NOTHING",
			tbaId, "Test Team", 0)
		if err != nil {
			return fmt.Errorf("failed to seed test team %s: %w", tbaId, err)
		}
	}
	return nil
}

// createTestUser creates a user with a unique username and returns the uuid.
func createTestUser(t *testing.T, db *sql.DB) User {
	t.Helper()

	store := NewSQLUserStore(db)
	ctx := context.Background()

	username := "testuser_" + randomString(8)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	require.NoError(t, err)
	userUuid, err := store.RegisterUser(ctx, username, string(passwordHash))
	require.NoError(t, err)

	user := User{
		UserUuid: userUuid,
		Username: username,
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Users WHERE UserUuid = $1", userUuid)
	})

	return user
}

// createTestDraft creates a draft owned by the given user.
func createTestDraft(t *testing.T, db *sql.DB, owner User) DraftModel {
	t.Helper()

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	draft := &DraftModel{
		DisplayName: "Test Draft " + randomString(8),
		Description: "Integration test draft",
		Owner:       owner,
		Status:      FILLING,
	}

	draftId, err := store.CreateDraft(ctx, draft)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Drafts WHERE Id = $1", draftId)
	})

	draft.Id = draftId
	return *draft
}

// createTestTeam inserts a team record for scoring tests.
func createTestTeam(t *testing.T, db *sql.DB, tbaId string) {
	t.Helper()

	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO Teams (tbaId, name, allianceScore) VALUES ($1, $2, $3) ON CONFLICT (tbaId) DO UPDATE SET name = EXCLUDED.name", tbaId, "Test Team", 10)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Teams WHERE tbaId = $1", tbaId)
	})
}

// randomString returns a simple random alphanumeric string for unique test data.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[randInt(len(letters))]
	}
	return string(b)
}

func randInt(max int) int {
	// Simple deterministic randomness is fine for test data uniqueness
	return int(timeNow().UnixNano()) % max
}

func timeNow() time.Time {
	return time.Now()
}
