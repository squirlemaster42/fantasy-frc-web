package model

import (
	"context"
	"os"
	"path/filepath"
	"server/database"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiKeyStore_Integration(t *testing.T) {
	err := godotenv.Load(filepath.Join("../", ".env"))
	if err != nil {
		t.Skipf("Skipping test: failed to load .env file %v", err)
	}

	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbIp := os.Getenv("DB_IP")
	dbName := os.Getenv("DB_NAME")

	if dbUsername == "" || dbPassword == "" || dbIp == "" || dbName == "" {
		t.Skip("Skipping test: database credentials not found in environment")
	}

	ctx := context.Background()
	db, err := database.RegisterDatabaseConnection(ctx, dbUsername, dbPassword, dbIp, dbName)
	require.NoError(t, err)
	defer db.Close()

	store := NewSQLApiKeyStore(db)
	userStore := NewSQLUserStore(db)

	t.Run("create and validate api key", func(t *testing.T) {
		userUuid, err := userStore.RegisterUser(ctx, "test_api_key_user", "test-password-12345")
		require.NoError(t, err)
		defer func() {
			_, _ = db.ExecContext(ctx, "DELETE FROM UserApiKeys WHERE UserUuid = $1", userUuid)
			_, _ = db.ExecContext(ctx, "DELETE FROM Users WHERE UserUuid = $1", userUuid)
		}()

		key, secret, err := store.CreateApiKey(ctx, userUuid, "Test Integration")
		require.NoError(t, err)
		require.NotNil(t, key)
		require.NotEmpty(t, secret)
		assert.Equal(t, userUuid, key.UserUuid)
		assert.Equal(t, "Test Integration", key.DisplayName)
		assert.False(t, key.Revoked)
		assert.Equal(t, StringArray{"full_access"}, key.Scopes)

		validatedUuid, err := store.ValidateApiKey(ctx, key.ClientId, secret)
		require.NoError(t, err)
		assert.Equal(t, userUuid, validatedUuid)

		_, err = store.ValidateApiKey(ctx, key.ClientId, "wrong-secret")
		assert.ErrorIs(t, err, ErrInvalidApiKey)

		keys, err := store.GetApiKeysForUser(ctx, userUuid)
		require.NoError(t, err)
		assert.NotEmpty(t, keys)

		err = store.RevokeApiKey(ctx, key.Id, userUuid)
		require.NoError(t, err)

		_, err = store.ValidateApiKey(ctx, key.ClientId, secret)
		assert.ErrorIs(t, err, ErrInvalidApiKey)
	})
}

