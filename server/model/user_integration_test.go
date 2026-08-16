package model

import (
	"context"
	"crypto"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestRegisterUser_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	username := "test_register_" + randomString(8)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	require.NoError(t, err)

	userUuid, err := store.RegisterUser(ctx, username, string(passwordHash))

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, userUuid)

	// Cleanup is handled by createTestUser helper, but here we register directly
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Users WHERE UserUuid = $1", userUuid)
	})

	taken, err := store.UsernameTaken(ctx, username)
	assert.NoError(t, err)
	assert.True(t, taken)

	storedHash, err := store.GetPasswordHashByUsername(ctx, username)
	assert.NoError(t, err)
	assert.Equal(t, string(passwordHash), storedHash)
}

func TestGetPasswordHashByUsername_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	passwordHash, err := store.GetPasswordHashByUsername(ctx, user.Username)
	assert.NoError(t, err)
	assert.NotEmpty(t, passwordHash)
	assert.True(t, strings.HasPrefix(passwordHash, "$2a$"), "password should be a bcrypt hash")
}

func TestSessionTokenFlow_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	sessionToken := "test-session-token-" + randomString(8)
	err := store.RegisterSession(ctx, user.UserUuid, sessionToken)
	require.NoError(t, err)

	valid, err := store.ValidateSessionToken(ctx, sessionToken)
	assert.NoError(t, err)
	assert.True(t, valid)

	foundUuid, err := store.GetUserBySessionToken(ctx, sessionToken)
	assert.NoError(t, err)
	assert.Equal(t, user.UserUuid, foundUuid)

	err = store.UnRegisterSession(ctx, sessionToken)
	assert.NoError(t, err)

	valid, err = store.ValidateSessionToken(ctx, sessionToken)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestInvalidateAllUserSessionsExcept_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	token1 := "token-keep-" + randomString(8)
	token2 := "token-discard-" + randomString(8)

	require.NoError(t, store.RegisterSession(ctx, user.UserUuid, token1))
	require.NoError(t, store.RegisterSession(ctx, user.UserUuid, token2))

	err := store.InvalidateAllUserSessionsExcept(ctx, user.UserUuid, token1)
	require.NoError(t, err)

	valid, err := store.ValidateSessionToken(ctx, token1)
	assert.NoError(t, err)
	assert.True(t, valid)

	valid, err = store.ValidateSessionToken(ctx, token2)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestUserIsAdmin_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	isAdmin, err := store.UserIsAdmin(ctx, user.UserUuid)
	assert.NoError(t, err)
	assert.False(t, isAdmin)
}

func TestUpdatePassword_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte("NewPassword456!"), bcrypt.MinCost)
	require.NoError(t, err)

	err = store.UpdatePassword(ctx, user.Username, string(newPasswordHash))
	require.NoError(t, err)

	storedHash, err := store.GetPasswordHashByUsername(ctx, user.Username)
	require.NoError(t, err)
	assert.Equal(t, string(newPasswordHash), storedHash)
}

func TestDiscordId_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	discordId, err := store.GetDiscordId(ctx, user.UserUuid)
	assert.NoError(t, err)
	assert.Empty(t, discordId)

	err = store.UpdateDiscordId(ctx, user.UserUuid, "12345678901234567")
	require.NoError(t, err)

	discordId, err = store.GetDiscordId(ctx, user.UserUuid)
	assert.NoError(t, err)
	assert.Equal(t, "12345678901234567", discordId)
}

func TestSearchUsers_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Searching for the exact username should find the user
	users, err := store.SearchUsers(ctx, user.Username, 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, users)

	found := false
	for _, u := range users {
		if u.UserUuid == user.UserUuid {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestGetUsername_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	username, err := store.GetUsername(ctx, user.UserUuid)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, username)
}

func TestGetUserUuidByUsername_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	foundUuid, err := store.GetUserUuidByUsername(ctx, user.Username)
	assert.NoError(t, err)
	assert.Equal(t, user.UserUuid, foundUuid)
}

func TestGetUserBySessionToken_ExpiredSession_Integration(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	user := createTestUser(t, db)

	sessionToken := "expired-token-" + randomString(8)
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	hashedToken := hasher.Sum(nil)

	_, err := db.ExecContext(ctx,
		"INSERT INTO UserSessions (userUuid, sessionToken, expirationTime) VALUES ($1, $2, now() - interval '1 day')",
		user.UserUuid, hashedToken)
	require.NoError(t, err)

	store := NewSQLUserStore(db)
	_, err = store.GetUserBySessionToken(ctx, sessionToken)
	assert.Error(t, err)

	valid, err := store.ValidateSessionToken(ctx, sessionToken)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestUsernameTaken_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLUserStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	taken, err := store.UsernameTaken(ctx, user.Username)
	assert.NoError(t, err)
	assert.True(t, taken)

	taken, err = store.UsernameTaken(ctx, "totally_unique_"+randomString(16))
	assert.NoError(t, err)
	assert.False(t, taken)
}

func TestRegisterUser_PasswordHashStoredAsProvided(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	store := NewSQLUserStore(db)

	username := "test_hash_" + randomString(8)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), 10)
	require.NoError(t, err)

	userUuid, err := store.RegisterUser(ctx, username, string(passwordHash))
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Users WHERE UserUuid = $1", userUuid)
	})

	var storedHash string
	err = db.QueryRowContext(ctx, "SELECT password FROM Users WHERE UserUuid = $1", userUuid).Scan(&storedHash)
	require.NoError(t, err)
	assert.Equal(t, string(passwordHash), storedHash)

	hashCost, err := bcrypt.Cost([]byte(storedHash))
	require.NoError(t, err)
	assert.Equal(t, 10, hashCost, "stored password hash should preserve the cost it was given")
}
