package model

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDraftsForUser_NewQuery_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	t.Run("returns owned drafts", func(t *testing.T) {
		owner := createTestUser(t, db)
		draft := createTestDraft(t, db, owner)

		drafts, err := store.GetDraftsForUser(ctx, owner.UserUuid)
		require.NoError(t, err)
		require.Len(t, drafts, 1)
		assert.Equal(t, draft.Id, drafts[0].Id)
		assert.Equal(t, draft.DisplayName, drafts[0].DisplayName)
	})

	t.Run("returns drafts where user is an accepted player", func(t *testing.T) {
		owner := createTestUser(t, db)
		draft := createTestDraft(t, db, owner)

		player := createTestUser(t, db)
		_, err := db.ExecContext(ctx, "INSERT INTO DraftPlayers (draftId, userUuid) VALUES ($1, $2)", draft.Id, player.UserUuid)
		require.NoError(t, err)

		drafts, err := store.GetDraftsForUser(ctx, player.UserUuid)
		require.NoError(t, err)
		require.Len(t, drafts, 1)
		assert.Equal(t, draft.Id, drafts[0].Id)
	})

	t.Run("does not return drafts where user has only a pending invite", func(t *testing.T) {
		owner := createTestUser(t, db)
		draft := createTestDraft(t, db, owner)

		invited := createTestUser(t, db)
		_, err := db.ExecContext(ctx,
			"INSERT INTO DraftInvites (draftId, invitingUserUuid, invitedUserUuid, sentTime, Status) VALUES ($1, $2, $3, $4, $5)",
			draft.Id, owner.UserUuid, invited.UserUuid, time.Now().UTC(), "pending")
		require.NoError(t, err)

		drafts, err := store.GetDraftsForUser(ctx, invited.UserUuid)
		require.NoError(t, err)
		assert.Empty(t, drafts)
	})

	t.Run("does not return other users' drafts", func(t *testing.T) {
		owner := createTestUser(t, db)
		_ = createTestDraft(t, db, owner)

		other := createTestUser(t, db)
		drafts, err := store.GetDraftsForUser(ctx, other.UserUuid)
		require.NoError(t, err)
		assert.Empty(t, drafts)
	})
}

func TestUserDraftNotificationPreferences_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	t.Run("returns empty map when no preferences exist", func(t *testing.T) {
		prefs, err := store.GetUserDraftNotificationPreferences(ctx, owner.UserUuid)
		require.NoError(t, err)
		assert.Empty(t, prefs)
	})

	t.Run("upserts and retrieves preferences", func(t *testing.T) {
		err := store.UpdateUserDraftNotificationPreference(ctx, owner.UserUuid, draft.Id, true, false)
		require.NoError(t, err)

		prefs, err := store.GetUserDraftNotificationPreferences(ctx, owner.UserUuid)
		require.NoError(t, err)
		require.Len(t, prefs, 1)

		pref, ok := prefs[draft.Id]
		require.True(t, ok)
		assert.Equal(t, owner.UserUuid, pref.UserUuid)
		assert.Equal(t, draft.Id, pref.DraftId)
		assert.True(t, pref.UpcomingMatch)
		assert.False(t, pref.PickTurn)
	})

	t.Run("upsert updates existing preferences", func(t *testing.T) {
		err := store.UpdateUserDraftNotificationPreference(ctx, owner.UserUuid, draft.Id, false, true)
		require.NoError(t, err)

		prefs, err := store.GetUserDraftNotificationPreferences(ctx, owner.UserUuid)
		require.NoError(t, err)

		pref, ok := prefs[draft.Id]
		require.True(t, ok)
		assert.False(t, pref.UpcomingMatch)
		assert.True(t, pref.PickTurn)
	})
}

func TestGetPlayerPickNotificationId_Integration(t *testing.T) {
	db := setupTestDB(t)
	draftStore := NewSQLDraftStore(db)
	discordStore := NewSQLDiscordStore(db)
	ctx := context.Background()

	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	player := createTestUser(t, db)
	_, err := db.ExecContext(ctx, "INSERT INTO DraftPlayers (draftId, userUuid) VALUES ($1, $2)", draft.Id, player.UserUuid)
	require.NoError(t, err)

	playerId, err := draftStore.GetDraftPlayerId(ctx, draft.Id, player.UserUuid)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, "UPDATE Users SET DiscordId = $1 WHERE UserUuid = $2", "12345678901234567", player.UserUuid)
	require.NoError(t, err)

	t.Run("returns invalid when pick turn notifications are disabled", func(t *testing.T) {
		err := draftStore.UpdateUserDraftNotificationPreference(ctx, player.UserUuid, draft.Id, false, false)
		require.NoError(t, err)

		discordId, err := discordStore.GetPlayerPickNotificationId(ctx, playerId)
		require.NoError(t, err)
		assert.False(t, discordId.Valid)
	})

	t.Run("returns discord id when pick turn notifications are enabled", func(t *testing.T) {
		err := draftStore.UpdateUserDraftNotificationPreference(ctx, player.UserUuid, draft.Id, false, true)
		require.NoError(t, err)

		discordId, err := discordStore.GetPlayerPickNotificationId(ctx, playerId)
		require.NoError(t, err)
		assert.True(t, discordId.Valid)
		assert.Equal(t, "12345678901234567", discordId.String)
	})
}

func TestGetDraftPickRows_WithPreferences_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	discordStore := NewSQLDiscordStore(db)
	ctx := context.Background()

	t.Run("returns row with upcoming match preference", func(t *testing.T) {
		owner := createTestUser(t, db)
		draft := createTestDraft(t, db, owner)

		teamId := "frc" + randomString(4)
		createTestTeam(t, db, teamId)

		_, err := db.ExecContext(ctx, "UPDATE Drafts SET DiscordWebhook = $1 WHERE Id = $2", "https://discord.com/api/webhooks/test", draft.Id)
		require.NoError(t, err)

		_, err = db.ExecContext(ctx, "UPDATE Users SET DiscordId = $1 WHERE UserUuid = $2", "12345678901234567", owner.UserUuid)
		require.NoError(t, err)

		playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
		require.NoError(t, err)

		pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
		require.NoError(t, err)

		pick := Pick{
			Id:       pickId,
			Player:   playerId,
			Pick:     sql.NullString{String: teamId, Valid: true},
			PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		}
		err = store.MakePick(ctx, pick)
		require.NoError(t, err)

		t.Run("false when upcoming match notifications are disabled", func(t *testing.T) {
			err := store.UpdateUserDraftNotificationPreference(ctx, owner.UserUuid, draft.Id, false, false)
			require.NoError(t, err)

			rows, err := store.GetDraftPickRows(ctx, []string{teamId})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.False(t, rows[0].WantsUpcomingMatch)
		})

		t.Run("true when upcoming match notifications are enabled", func(t *testing.T) {
			err := store.UpdateUserDraftNotificationPreference(ctx, owner.UserUuid, draft.Id, true, false)
			require.NoError(t, err)

			rows, err := store.GetDraftPickRows(ctx, []string{teamId})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.True(t, rows[0].WantsUpcomingMatch)

			// Also verify the raw player Discord ID is still returned separately.
			discordId, err := discordStore.GetPlayerDiscordId(ctx, playerId)
			require.NoError(t, err)
			assert.True(t, discordId.Valid)
			assert.Equal(t, "12345678901234567", discordId.String)
		})
	})
}

