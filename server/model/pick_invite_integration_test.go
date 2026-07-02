package model

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvitePlayer_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	invited := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	inviteId, err := store.InvitePlayer(ctx, draft.Id, owner.UserUuid, invited.UserUuid)
	require.NoError(t, err)
	assert.NotZero(t, inviteId)

	invite, err := store.GetInvite(ctx, inviteId)
	require.NoError(t, err)
	assert.Equal(t, draft.Id, invite.DraftId)
	assert.Equal(t, invited.UserUuid, invite.InvitedUserUuid)
}

func TestAcceptInvite_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	invited := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	inviteId, err := store.InvitePlayer(ctx, draft.Id, owner.UserUuid, invited.UserUuid)
	require.NoError(t, err)

	draftId, userUuid, err := store.AcceptInvite(ctx, inviteId)
	require.NoError(t, err)
	assert.Equal(t, draft.Id, draftId)
	assert.Equal(t, invited.UserUuid, userUuid)

	// AcceptInvite only marks the invite accepted; the caller must add the player
	err = store.AddPlayerToDraft(ctx, draft.Id, invited.UserUuid)
	require.NoError(t, err)

	// User should now be a player in the draft
	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, invited.UserUuid)
	require.NoError(t, err)
	assert.NotZero(t, playerId)
}

func TestGetInvites_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	invited := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	_, err := store.InvitePlayer(ctx, draft.Id, owner.UserUuid, invited.UserUuid)
	require.NoError(t, err)

	invites, err := store.GetInvites(ctx, invited.UserUuid)
	require.NoError(t, err)
	assert.Len(t, invites, 1)
	assert.Equal(t, draft.DisplayName, invites[0].DraftName)
}

func TestGetInvite_NotFound_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	_, err := store.GetInvite(ctx, -1)
	assert.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestMakePickAvailableAndGetCurrentPick_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	now := time.Now().UTC()
	expiration := now.Add(time.Hour)
	pickId, err := store.MakePickAvailable(ctx, playerId, now, expiration)
	require.NoError(t, err)
	assert.NotZero(t, pickId)

	pick, err := store.GetCurrentPick(ctx, draft.Id)
	require.NoError(t, err)
	assert.Equal(t, pickId, pick.Id)
	assert.Equal(t, playerId, pick.Player)
	assert.WithinDuration(t, expiration, pick.ExpirationTime, time.Second)
}

func TestMakePick_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	pick := Pick{
		Id:       pickId,
		Player:   playerId,
		Pick:     sql.NullString{String: "frc254", Valid: true},
		PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}
	err = store.MakePick(ctx, pick)
	require.NoError(t, err)

	picks, err := store.GetPicks(ctx, draft.Id)
	require.NoError(t, err)
	require.Len(t, picks, 1)
	assert.Equal(t, "frc254", picks[0].Pick.String)
	assert.True(t, picks[0].PickTime.Valid)
}

func TestHasBeenPicked_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	picked, err := store.HasBeenPicked(ctx, draft.Id, "frc254")
	require.NoError(t, err)
	assert.False(t, picked)

	pick := Pick{
		Id:       pickId,
		Player:   playerId,
		Pick:     sql.NullString{String: "frc254", Valid: true},
		PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}
	err = store.MakePick(ctx, pick)
	require.NoError(t, err)

	picked, err = store.HasBeenPicked(ctx, draft.Id, "frc254")
	require.NoError(t, err)
	assert.True(t, picked)
}

func TestSkipPick_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	err = store.SkipPick(ctx, pickId)
	require.NoError(t, err)

	picks, err := store.GetPicks(ctx, draft.Id)
	require.NoError(t, err)
	require.Len(t, picks, 1)
	assert.True(t, picks[0].Skipped)
}

func TestUpdatePickExpirationTime_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	newExpiration := time.Now().UTC().Add(2 * time.Hour)
	err = store.UpdatePickExpirationTime(ctx, pickId, newExpiration)
	require.NoError(t, err)

	pick, err := store.GetCurrentPick(ctx, draft.Id)
	require.NoError(t, err)
	assert.WithinDuration(t, newExpiration, pick.ExpirationTime, time.Second)
}

func TestDeletePick_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	err = store.DeletePick(ctx, pickId)
	require.NoError(t, err)

	_, err = store.GetCurrentPick(ctx, draft.Id)
	assert.Error(t, err)
}

func TestResetPick_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	pick := Pick{
		Id:       pickId,
		Player:   playerId,
		Pick:     sql.NullString{String: "frc254", Valid: true},
		PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}
	err = store.MakePick(ctx, pick)
	require.NoError(t, err)

	newExpiration := time.Now().UTC().Add(2 * time.Hour)
	err = store.ResetPick(ctx, pickId, newExpiration)
	require.NoError(t, err)

	picks, err := store.GetPicks(ctx, draft.Id)
	require.NoError(t, err)
	require.Len(t, picks, 1)
	assert.False(t, picks[0].Pick.Valid)
	assert.False(t, picks[0].PickTime.Valid)
	assert.False(t, picks[0].Skipped)
	assert.WithinDuration(t, newExpiration, picks[0].ExpirationTime, time.Second)
}

func TestGetPreviousPick_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	pickId1, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-time.Hour))
	require.NoError(t, err)

	pickId2, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	previous, err := store.GetPreviousPick(ctx, draft.Id, pickId2)
	require.NoError(t, err)
	assert.Equal(t, pickId1, previous.Id)
}

func TestShouldSkipPickAndMarkShouldSkipPick_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	shouldSkip, err := store.ShouldSkipPick(ctx, playerId)
	require.NoError(t, err)
	assert.False(t, shouldSkip)

	err = store.MarkShouldSkipPick(ctx, playerId, true)
	require.NoError(t, err)

	shouldSkip, err = store.ShouldSkipPick(ctx, playerId)
	require.NoError(t, err)
	assert.True(t, shouldSkip)
}

func TestGetNumPlayersInInvitedDraft_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	invited := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	inviteId, err := store.InvitePlayer(ctx, draft.Id, owner.UserUuid, invited.UserUuid)
	require.NoError(t, err)

	numPlayers, err := store.GetNumPlayersInInvitedDraft(ctx, inviteId)
	require.NoError(t, err)
	assert.Equal(t, 1, numPlayers)
}

func TestAddPlayerToDraft_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	newPlayer := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	err := store.AddPlayerToDraft(ctx, draft.Id, newPlayer.UserUuid)
	require.NoError(t, err)

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, newPlayer.UserUuid)
	require.NoError(t, err)
	assert.NotZero(t, playerId)
}

func TestRandomizePickOrder_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	player2 := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	err := store.AddPlayerToDraft(ctx, draft.Id, player2.UserUuid)
	require.NoError(t, err)

	err = store.UpdateDraftStatus(ctx, draft.Id, WAITING_TO_START)
	require.NoError(t, err)

	err = store.RandomizePickOrder(ctx, draft.Id)
	require.NoError(t, err)

	loaded, err := store.GetDraft(ctx, draft.Id)
	require.NoError(t, err)
	require.Len(t, loaded.Players, 2)

	ordersSet := 0
	for _, player := range loaded.Players {
		if player.PlayerOrder.Valid {
			ordersSet++
		}
	}
	assert.Equal(t, 2, ordersSet)
}

func TestNextPick_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	player2 := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	err := store.AddPlayerToDraft(ctx, draft.Id, player2.UserUuid)
	require.NoError(t, err)

	err = store.RandomizePickOrder(ctx, draft.Id)
	require.NoError(t, err)

	nextPlayer, err := store.NextPick(ctx, draft.Id)
	require.NoError(t, err)
	assert.NotZero(t, nextPlayer.Id)
}

func TestGetDraftScore_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	teamId := "frc" + randomString(4)
	createTestTeam(t, db, teamId)

	pick := Pick{
		Id:       pickId,
		Player:   playerId,
		Pick:     sql.NullString{String: teamId, Valid: true},
		PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}
	err = store.MakePick(ctx, pick)
	require.NoError(t, err)

	scores, err := store.GetDraftScore(ctx, draft.Id)
	require.NoError(t, err)
	require.Len(t, scores, 1)
	assert.Equal(t, 10, scores[0].Score)
}

func TestGetDraftPickRows_Integration(t *testing.T) {
	db := setupTestDB(t)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	// With no teams picked, should return empty
	rows, err := store.GetDraftPickRows(ctx, []string{"frc254"})
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestGetDraftPlayerUser_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	user, err := store.GetDraftPlayerUser(ctx, playerId)
	require.NoError(t, err)
	assert.Equal(t, owner.UserUuid, user.UserUuid)
	assert.Equal(t, owner.Username, user.Username)
}

func TestGetDraftPlayerId_NotFound_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	_, err := store.GetDraftPlayerId(ctx, draft.Id, uuid.New())
	assert.Error(t, err)
}
