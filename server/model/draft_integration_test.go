package model

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDraft_Integration(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)

	draft := createTestDraft(t, db, user)

	assert.NotZero(t, draft.Id)
	assert.Equal(t, FILLING, draft.Status)
	assert.Equal(t, user.UserUuid, draft.Owner.UserUuid)
}

func TestGetDraft_Integration(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	draft := createTestDraft(t, db, user)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	loaded, err := store.GetDraft(ctx, draft.Id)
	require.NoError(t, err)
	assert.Equal(t, draft.Id, loaded.Id)
	assert.Equal(t, draft.DisplayName, loaded.DisplayName)
	assert.Equal(t, draft.Status, loaded.Status)
}

func TestGetDraftsForUser_Integration(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	draft := createTestDraft(t, db, user)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	drafts, err := store.GetDraftsForUser(ctx, user.UserUuid)
	require.NoError(t, err)

	found := false
	for _, d := range drafts {
		if d.Id == draft.Id {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestUpdateDraft_Integration(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	draft := createTestDraft(t, db, user)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	draft.DisplayName = "Updated Name " + randomString(8)
	draft.Description = "Updated description"
	draft.Interval = 7200

	err := store.UpdateDraft(ctx, &draft)
	require.NoError(t, err)

	loaded, err := store.GetDraft(ctx, draft.Id)
	require.NoError(t, err)
	assert.Equal(t, draft.DisplayName, loaded.DisplayName)
	assert.Equal(t, draft.Description, loaded.Description)
	assert.Equal(t, draft.Interval, loaded.Interval)
}

func TestUpdateDraftStatus_Integration(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	draft := createTestDraft(t, db, user)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	err := store.UpdateDraftStatus(ctx, draft.Id, WAITING_TO_START)
	require.NoError(t, err)

	loaded, err := store.GetDraft(ctx, draft.Id)
	require.NoError(t, err)
	assert.Equal(t, WAITING_TO_START, loaded.Status)
}

func TestGetDraftsByName_Integration(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	draft := createTestDraft(t, db, user)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	// Search by a unique substring of the display name
	searchTerm := draft.DisplayName[len("Test Draft "):]
	drafts, err := store.GetDraftsByName(ctx, searchTerm)
	require.NoError(t, err)

	found := false
	for _, d := range drafts {
		if d.Id == draft.Id {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestGetDraftsInStatus_Integration(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	draft := createTestDraft(t, db, user)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	drafts, err := store.GetDraftsInStatus(ctx, FILLING)
	require.NoError(t, err)

	found := false
	for _, id := range drafts {
		if id == draft.Id {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestCancelOutstandingInvites_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	invited := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	inviteId, err := store.InvitePlayer(ctx, draft.Id, owner.UserUuid, invited.UserUuid)
	require.NoError(t, err)

	err = store.CancelOutstandingInvites(ctx, draft.Id)
	require.NoError(t, err)

	var status string
	err = db.QueryRowContext(ctx, "SELECT Status FROM DraftInvites WHERE Id = $1", inviteId).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "canceled", status)
}

func TestGetDraftsForUser_MultipleDrafts_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	invited := createTestUser(t, db)

	draftA := createTestDraft(t, db, owner)
	draftB := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	_, err := store.InvitePlayer(ctx, draftB.Id, owner.UserUuid, invited.UserUuid)
	require.NoError(t, err)

	drafts, err := store.GetDraftsForUser(ctx, owner.UserUuid)
	require.NoError(t, err)
	require.Len(t, drafts, 2)

	draftIds := make(map[int]bool)
	for _, d := range drafts {
		draftIds[d.Id] = true
		if d.Id == draftA.Id {
			require.Len(t, d.Players, 1)
			assert.False(t, d.Players[0].Pending)
		}
		if d.Id == draftB.Id {
			require.Len(t, d.Players, 2)
		}
	}
	assert.True(t, draftIds[draftA.Id])
	assert.True(t, draftIds[draftB.Id])

	invitedDrafts, err := store.GetDraftsForUser(ctx, invited.UserUuid)
	require.NoError(t, err)
	require.Len(t, invitedDrafts, 1)
	assert.Equal(t, draftB.Id, invitedDrafts[0].Id)
	require.Len(t, invitedDrafts[0].Players, 2)
	pendingCount := 0
	for _, p := range invitedDrafts[0].Players {
		if p.Pending {
			pendingCount++
		}
	}
	assert.Equal(t, 1, pendingCount)
}

func TestGetDraftsForUser_PickingStatus_Integration(t *testing.T) {
	db := setupTestDB(t)
	owner := createTestUser(t, db)
	draft := createTestDraft(t, db, owner)

	store := NewSQLDraftStore(db)
	ctx := context.Background()

	err := store.UpdateDraftStatus(ctx, draft.Id, PICKING)
	require.NoError(t, err)

	playerId, err := store.GetDraftPlayerId(ctx, draft.Id, owner.UserUuid)
	require.NoError(t, err)

	now := time.Now().UTC()
	_, err = store.MakePickAvailable(ctx, playerId, now, now.Add(time.Hour))
	require.NoError(t, err)

	drafts, err := store.GetDraftsForUser(ctx, owner.UserUuid)
	require.NoError(t, err)
	require.Len(t, drafts, 1)
	assert.Equal(t, PICKING, drafts[0].Status)
	assert.Equal(t, playerId, drafts[0].NextPick.Id)
	assert.Equal(t, owner.UserUuid, drafts[0].NextPick.User.UserUuid)
	assert.Equal(t, owner.Username, drafts[0].NextPick.User.Username)
}
