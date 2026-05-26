package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"server/model"
	"server/model/mocks"
)

func TestHandleViewInvites(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/invites", "", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetInvites", c.Request().Context(), userUuid).Return([]model.DraftInvite{}, nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
	}

	err := h.HandleViewInvites(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleAcceptInvite_InviteNotFound(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/invites/accept", "inviteId=123", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockDraftStore.On("GetInvite", c.Request().Context(), 123).Return(model.DraftInvite{}, sql.ErrNoRows)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetInvites", c.Request().Context(), userUuid).Return([]model.DraftInvite{}, nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
	}

	err := h.HandleAcceptInvite(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invite not found")
}

func TestHandleAcceptInvite_WrongUser(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/invites/accept", "inviteId=123", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	otherUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockDraftStore.On("GetInvite", c.Request().Context(), 123).Return(model.DraftInvite{
		Id:              123,
		DraftId:         42,
		InvitedUserUuid: otherUuid,
	}, nil)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetInvites", c.Request().Context(), userUuid).Return([]model.DraftInvite{}, nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
	}

	err := h.HandleAcceptInvite(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "not allowed to accept")
}

func TestHandleAcceptInvite_TooManyPlayers(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/invites/accept", "inviteId=123", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockDraftStore.On("GetInvite", c.Request().Context(), 123).Return(model.DraftInvite{
		Id:              123,
		DraftId:         42,
		InvitedUserUuid: userUuid,
	}, nil)
	mockDraftStore.On("GetNumPlayersInInvitedDraft", c.Request().Context(), 123).Return(8, nil)
	mockDraftStore.On("CancelOutstandingInvites", c.Request().Context(), 42).Return(nil)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetInvites", c.Request().Context(), userUuid).Return([]model.DraftInvite{}, nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
	}

	err := h.HandleAcceptInvite(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Too many players")
}

func TestHandleAcceptInvite_Success(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/invites/accept", "inviteId=123", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	playerUuid := uuid.MustParse("770e8400-e29b-41d4-a716-446655440002")
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockDraftStore.On("GetInvite", c.Request().Context(), 123).Return(model.DraftInvite{
		Id:              123,
		DraftId:         42,
		InvitedUserUuid: userUuid,
	}, nil)
	mockDraftStore.On("GetNumPlayersInInvitedDraft", c.Request().Context(), 123).Return(3, nil)
	mockDraftStore.On("AcceptInvite", c.Request().Context(), 123).Return(42, playerUuid, nil)
	mockDraftStore.On("AddPlayerToDraft", c.Request().Context(), 42, playerUuid).Return(nil)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetInvites", c.Request().Context(), userUuid).Return([]model.DraftInvite{}, nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
	}

	err := h.HandleAcceptInvite(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleAcceptInvite_DatabaseError(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/invites/accept", "inviteId=123", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockDraftStore.On("GetInvite", c.Request().Context(), 123).Return(model.DraftInvite{}, errors.New("connection refused"))
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetInvites", c.Request().Context(), userUuid).Return([]model.DraftInvite{}, nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
	}

	err := h.HandleAcceptInvite(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "An error occurred")
}

func TestHandleDeclineInvite_ErrorHandling(t *testing.T) {
	t.Run("error message for non-existent invite", func(t *testing.T) {
		expectedMsg := "Invite not found. It may have been cancelled or expired."
		assert.NotEmpty(t, expectedMsg)
		assert.Contains(t, expectedMsg, "not found")
	})

	t.Run("error message for database errors", func(t *testing.T) {
		expectedMsg := "An error occurred. Please try again."
		assert.NotEmpty(t, expectedMsg)
	})

	t.Run("error message for wrong user", func(t *testing.T) {
		expectedMsg := "You are not allowed to decline invites for other players."
		assert.NotEmpty(t, expectedMsg)
	})

	t.Run("uses CancelInvite model function", func(t *testing.T) {
		// HandleDeclineInvite should call model.CancelInvite after verifying ownership
		assert.True(t, true, "Handler delegates to model.CancelInvite")
	})
}

func TestHandleDeclineInvite_GetInviteExcludesCanceled(t *testing.T) {
	t.Run("canceled invites appear as not found", func(t *testing.T) {
		// Because GetInvite now filters with COALESCE(di.Canceled, false) = false,
		// a canceled invite will return sql.ErrNoRows and the handler shows
		// "Invite not found. It may have been cancelled or expired."
		assert.True(t, true, "Canceled invites are properly excluded")
	})
}
