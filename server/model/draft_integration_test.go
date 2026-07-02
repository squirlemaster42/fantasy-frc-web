package model

import (
	"context"
	"testing"

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

	var canceled bool
	err = db.QueryRowContext(ctx, "SELECT COALESCE(Canceled, false) FROM DraftInvites WHERE Id = $1", inviteId).Scan(&canceled)
	require.NoError(t, err)
	assert.True(t, canceled)
}
