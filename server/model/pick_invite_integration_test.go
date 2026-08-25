package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"server/database"

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

func TestGetOverallLeaderboard_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	userA := createTestUser(t, db)
	draftA := createTestDraft(t, db, userA)

	userB := createTestUser(t, db)
	draftB := createTestDraft(t, db, userB)

	makePick := func(t *testing.T, draft DraftModel, user User, tbaId string) {
		t.Helper()
		playerId, err := store.GetDraftPlayerId(ctx, draft.Id, user.UserUuid)
		require.NoError(t, err)
		pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
		require.NoError(t, err)
		pick := Pick{
			Id:       pickId,
			Player:   playerId,
			Pick:     sql.NullString{String: tbaId, Valid: true},
			PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		}
		err = store.MakePick(ctx, pick)
		require.NoError(t, err)
	}

	teamA := "frc" + randomString(4)
	teamB := "frc" + randomString(4)
	createTestTeam(t, db, teamA)
	createTestTeam(t, db, teamB)

	makePick(t, draftA, userA, teamA)
	makePick(t, draftB, userB, teamB)

	// Paginate through leaderboard to find new entries
	var userAEntry *LeaderboardEntry
	var userBEntry *LeaderboardEntry
	for pageNum := 1; pageNum <= 100; pageNum++ {
		page, err := store.GetOverallLeaderboard(ctx, pageNum, 25)
		require.NoError(t, err)
		if len(page.Entries) == 0 {
			break
		}
		assert.GreaterOrEqual(t, page.Total, 2)
		for i, entry := range page.Entries {
			if entry.User.UserUuid == userA.UserUuid && entry.DraftId == draftA.Id {
				userAEntry = &page.Entries[i]
			}
			if entry.User.UserUuid == userB.UserUuid && entry.DraftId == draftB.Id {
				userBEntry = &page.Entries[i]
			}
		}
		if userAEntry != nil && userBEntry != nil {
			break
		}
	}
	require.NotNil(t, userAEntry, "Entry for userA in draftA not found")
	require.NotNil(t, userBEntry, "Entry for userB in draftB not found")
	assert.Equal(t, 10, userAEntry.Score, "userA should have score 10 from allianceScore")
	assert.Equal(t, 10, userBEntry.Score, "userB should have score 10 from allianceScore")
	assert.Len(t, userAEntry.Picks, 1)
	assert.Len(t, userBEntry.Picks, 1)
	assert.Equal(t, draftA.DisplayName, userAEntry.DraftName)
	assert.Equal(t, draftB.DisplayName, userBEntry.DraftName)
}

func TestGetOverallLeaderboard_Pagination_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	const numEntries = 5
	for i := 0; i < numEntries; i++ {
		user := createTestUser(t, db)
		draft := createTestDraft(t, db, user)
		team := "frc" + randomString(4)
		createTestTeam(t, db, team)

		playerId, err := store.GetDraftPlayerId(ctx, draft.Id, user.UserUuid)
		require.NoError(t, err)
		pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
		require.NoError(t, err)
		pick := Pick{
			Id:       pickId,
			Player:   playerId,
			Pick:     sql.NullString{String: team, Valid: true},
			PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		}
		err = store.MakePick(ctx, pick)
		require.NoError(t, err)
	}

	// Get baseline count before our entries
	baseline, err := store.GetOverallLeaderboard(ctx, 1, 500)
	require.NoError(t, err)
	expectedTotal := baseline.Total

	// Page 1 with perPage=2
	page1, err := store.GetOverallLeaderboard(ctx, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, expectedTotal, page1.Total)
	assert.Equal(t, 1, page1.CurrentPage)
	assert.Equal(t, 2, page1.PerPage)
	expectedPages := (expectedTotal + 1) / 2
	assert.Equal(t, expectedPages, page1.TotalPages)
	assert.Equal(t, min(2, expectedTotal), len(page1.Entries))

	// Last page should have the remainder
	lastPage, err := store.GetOverallLeaderboard(ctx, expectedPages, 2)
	require.NoError(t, err)
	assert.Equal(t, expectedPages, lastPage.CurrentPage)
	remainder := expectedTotal % 2
	if remainder == 0 {
		assert.Len(t, lastPage.Entries, 2)
	} else {
		assert.Len(t, lastPage.Entries, remainder)
	}

	// Page beyond last should clamp
	beyond, err := store.GetOverallLeaderboard(ctx, 999, 2)
	require.NoError(t, err)
	assert.Equal(t, expectedPages, beyond.CurrentPage)

	// Page 0 should clamp to page 1
	page0, err := store.GetOverallLeaderboard(ctx, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, 1, page0.CurrentPage)
}

func TestGetOverallLeaderboard_NoPicks_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	createTestDraft(t, db, user)

	page, err := store.GetOverallLeaderboard(ctx, 1, 25)
	require.NoError(t, err)
	// Function should not error; verify page structure is valid
	assert.GreaterOrEqual(t, 1, page.CurrentPage)
	assert.GreaterOrEqual(t, page.TotalPages, 1)
	assert.Equal(t, 25, page.PerPage)
}

func TestGetOverallLeaderboard_MultiplePicks_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	user := createTestUser(t, db)
	draft := createTestDraft(t, db, user)

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, user.UserUuid)
	require.NoError(t, err)

	teamA := "frc" + randomString(4)
	teamB := "frc" + randomString(4)
	createTestTeam(t, db, teamA)
	createTestTeam(t, db, teamB)

	for _, team := range []string{teamA, teamB} {
		pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
		require.NoError(t, err)
		pick := Pick{
			Id:       pickId,
			Player:   playerId,
			Pick:     sql.NullString{String: team, Valid: true},
			PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		}
		err = store.MakePick(ctx, pick)
		require.NoError(t, err)
	}

	// Our user should have a single entry with score 20 (2 * 10 allianceScore)
	page, err := store.GetOverallLeaderboard(ctx, 1, 500)
	require.NoError(t, err)
	var foundEntry *LeaderboardEntry
	for i, entry := range page.Entries {
		if entry.User.UserUuid == user.UserUuid && entry.DraftId == draft.Id {
			foundEntry = &page.Entries[i]
			break
		}
	}
	require.NotNil(t, foundEntry, "Entry for test user not found")
	assert.Equal(t, 20, foundEntry.Score, "Two picks with allianceScore=10 should sum to 20")
	assert.Len(t, foundEntry.Picks, 2)
}

func setupPickingDraft(t *testing.T, db *sql.DB, store DraftStore, numPlayers int) (DraftModel, []User) {
	t.Helper()

	ctx := context.Background()
	owner := createTestUser(t, db)
	users := []User{owner}
	draft := createTestDraft(t, db, owner)

	for i := 1; i < numPlayers; i++ {
		user := createTestUser(t, db)
		users = append(users, user)
		inviteId, err := store.InvitePlayer(ctx, draft.Id, owner.UserUuid, user.UserUuid)
		require.NoError(t, err)
		_, _, err = store.AcceptInvite(ctx, inviteId)
		require.NoError(t, err)
		err = store.AddPlayerToDraft(ctx, draft.Id, user.UserUuid)
		require.NoError(t, err)
	}

	err := store.RandomizePickOrder(ctx, draft.Id)
	require.NoError(t, err)
	err = store.UpdateDraftStatus(ctx, draft.Id, PICKING)
	require.NoError(t, err)

	return draft, users
}

func TestRunInTransaction_MakePickAndCreateNext_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	draft, users := setupPickingDraft(t, db, store, 3)

	team := "frc" + randomString(4)
	createTestTeam(t, db, team)

	firstPlayerId, err := store.GetDraftPlayerId(ctx, draft.Id, users[0].UserUuid)
	require.NoError(t, err)

	// Create the first available pick
	firstPickId, err := store.MakePickAvailable(ctx, firstPlayerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	pick := Pick{
		Id:       firstPickId,
		Player:   firstPlayerId,
		Pick:     sql.NullString{String: team, Valid: true},
		PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}

	// Determine the next pick using the same logic as the actor
	loadedDraft, err := store.GetDraft(ctx, draft.Id)
	require.NoError(t, err)
	nextPickPlayer, err := DetermineNextPick(loadedDraft.Players, loadedDraft.Picks)
	require.NoError(t, err)

	// Make the pick and advance atomically under a transaction
	err = store.RunInTransaction(ctx, func(tx database.DBTX) error {
		storeTx := store.WithTx(tx)
		if err := storeTx.MakePick(ctx, pick); err != nil {
			return err
		}
		_, err := storeTx.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
		return err
	})
	require.NoError(t, err)

	// Verify the original pick was recorded
	loadedDraft, err = store.GetDraft(ctx, draft.Id)
	require.NoError(t, err)

	madePick := false
	for _, p := range loadedDraft.Picks {
		if p.Id == firstPickId && p.Pick.String == team {
			madePick = true
		}
	}
	assert.True(t, madePick, "Original pick should be recorded")

	// Verify a new available pick was created
	currentPick, err := store.GetCurrentPick(ctx, draft.Id)
	require.NoError(t, err)
	assert.NotEqual(t, firstPickId, currentPick.Id, "A new pick should be available after advancing")
	assert.Equal(t, nextPickPlayer.Id, currentPick.Player, "Next pick should be assigned to the computed next player")
}

func TestRunInTransaction_FinalPick_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	draft, users := setupPickingDraft(t, db, store, 1)

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, users[0].UserUuid)
	require.NoError(t, err)

	// Create PicksPerDraft pick rows directly, marking all but the last as already made
	availablePickId := 0
	for i := 0; i < PicksPerDraft; i++ {
		pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
		require.NoError(t, err)
		if i < 63 {
			team := fmt.Sprintf("frc%02d", i)
			_, err = db.ExecContext(ctx, "INSERT INTO Teams (tbaId, name, allianceScore) VALUES ($1, $2, $3) ON CONFLICT (tbaId) DO NOTHING", team, "Test Team", 0)
			require.NoError(t, err)
			pick := Pick{
				Id:       pickId,
				Player:   playerId,
				Pick:     sql.NullString{String: team, Valid: true},
				PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
			}
			err = store.MakePick(ctx, pick)
			require.NoError(t, err)
		} else {
			availablePickId = pickId
		}
	}

	team := "frcfinal"
	_, err = db.ExecContext(ctx, "INSERT INTO Teams (tbaId, name, allianceScore) VALUES ($1, $2, $3) ON CONFLICT (tbaId) DO NOTHING", team, "Test Team", 0)
	require.NoError(t, err)

	pick := Pick{
		Id:       availablePickId,
		Player:   playerId,
		Pick:     sql.NullString{String: team, Valid: true},
		PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}

	// Make the final pick and transition the draft status under a transaction
	err = store.RunInTransaction(ctx, func(tx database.DBTX) error {
		storeTx := store.WithTx(tx)
		if err := storeTx.MakePick(ctx, pick); err != nil {
			return err
		}
		return storeTx.UpdateDraftStatus(ctx, draft.Id, TEAMS_PLAYING)
	})
	require.NoError(t, err)

	// Verify the final pick was recorded and status transitioned
	loadedDraft, err := store.GetDraft(ctx, draft.Id)
	require.NoError(t, err)
	assert.Equal(t, TEAMS_PLAYING, loadedDraft.Status)

	madeFinalPick := false
	for _, p := range loadedDraft.Picks {
		if p.Id == availablePickId && p.Pick.String == team {
			madeFinalPick = true
			break
		}
	}
	assert.True(t, madeFinalPick, "Final pick should be recorded")
}

func TestRunInTransaction_RollsBackOnError_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	draft, users := setupPickingDraft(t, db, store, 1)

	team := "frc" + randomString(4)
	createTestTeam(t, db, team)

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, users[0].UserUuid)
	require.NoError(t, err)

	pickId, err := store.MakePickAvailable(ctx, playerId, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	pick := Pick{
		Id:       pickId,
		Player:   playerId,
		Pick:     sql.NullString{String: team, Valid: true},
		PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}

	// Pass an invalid nextPickPlayerId to force the advance step to fail
	err = store.RunInTransaction(ctx, func(tx database.DBTX) error {
		storeTx := store.WithTx(tx)
		if err := storeTx.MakePick(ctx, pick); err != nil {
			return err
		}
		_, err := storeTx.MakePickAvailable(ctx, -1, time.Now().UTC(), time.Now().UTC().Add(time.Hour))
		return err
	})
	assert.Error(t, err)

	// Verify the pick was NOT recorded because the transaction rolled back
	loadedDraft, err := store.GetDraft(ctx, draft.Id)
	require.NoError(t, err)
	for _, p := range loadedDraft.Picks {
		if p.Id == pickId {
			assert.False(t, p.Pick.Valid, "Pick should not have been recorded after rollback")
		}
	}
}

func TestCreateDraft_RollsBackOnError_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	// Use a user UUID that does not exist in the Users table so the owner player insert fails.
	draft := &DraftModel{
		DisplayName: "Rollback Test Draft",
		Description: "Should rollback",
		Owner:       User{UserUuid: uuid.New(), Username: "nonexistent"},
		Status:      FILLING,
	}

	_, err := store.CreateDraft(ctx, draft)
	assert.Error(t, err)

	// Verify no draft row was left behind.
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM Drafts WHERE DisplayName = $1", draft.DisplayName).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Draft row should have been rolled back")
}

func TestAcceptInvite_RollsBackOnError_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLDraftStore(db)
	ctx := context.Background()

	owner := createTestUser(t, db)
	invited := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	inviteId, err := store.InvitePlayer(ctx, draft.Id, owner.UserUuid, invited.UserUuid)
	require.NoError(t, err)

	// Accepting the invite should fail when AddPlayerToDraft references a non-existent user,
	// and the transaction should roll back so the invite remains pending.
	err = store.RunInTransaction(ctx, func(tx database.DBTX) error {
		storeTx := store.WithTx(tx)
		if err := storeTx.LockDraft(ctx, draft.Id); err != nil {
			return err
		}
		numPlayers, err := storeTx.GetNumPlayersInDraft(ctx, draft.Id)
		if err != nil {
			return err
		}
		if numPlayers >= DraftPlayerCount {
			return errors.New("too many players")
		}
		draftId, _, err := storeTx.AcceptInvite(ctx, inviteId)
		if err != nil {
			return err
		}
		// Force a foreign-key violation so the transaction rolls back.
		return storeTx.AddPlayerToDraft(ctx, draftId, uuid.New())
	})
	assert.Error(t, err)

	// Verify the invite is still pending after rollback.
	var status string
	err = db.QueryRowContext(ctx, "SELECT Status FROM DraftInvites WHERE Id = $1", inviteId).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "pending", status)
}
