package apihandler

import (
	"encoding/json"
	"errors"
	"net/http"
	apimodel "server/api/model"
	"server/model"
	"server/model/mocks"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	mocklib "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"uuid"
)

func setupInvitesTest() (*Handler, *mocks.MockDraftStore, *echo.Echo) {
	draftStore := new(mocks.MockDraftStore)
	handler := NewHandler(draftStore, nil, nil, nil, nil)
	e := echo.New()
	return handler, draftStore, e
}

func TestListInvites_Success(t *testing.T) {
	handler, draftStore, e := setupInvitesTest()
	userUuid := uuid.New()
	invites := []model.DraftInvite{
		{
			Id:               1,
			DraftId:          1,
			DraftName:        "Draft One",
			InvitedUserUuid:  userUuid,
			InvitingUserUuid: uuid.New(),
			SentTime:         time.Now().UTC(),
			Status:           "pending",
		},
	}

	draftStore.On("GetInvites", mocklib.Anything, userUuid).Return(invites, nil)

	c := newDraftContext(t.Context(), e, userUuid, http.MethodGet, "/api/v1/invites", nil)
	err := handler.ListInvites(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, responseRecorder(c).Code)

	var resp []apimodel.InviteResponse
	require.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	assert.Equal(t, 1, resp[0].Id)
	assert.Equal(t, "Draft One", resp[0].DraftName)
	assert.Equal(t, "pending", resp[0].Status)
}

func TestListInvites_StoreError(t *testing.T) {
	handler, draftStore, e := setupInvitesTest()
	userUuid := uuid.New()

	draftStore.On("GetInvites", mocklib.Anything, userUuid).Return(nil, errors.New("db error"))

	c := newDraftContext(t.Context(), e, userUuid, http.MethodGet, "/api/v1/invites", nil)
	err := handler.ListInvites(c)

	require.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, responseRecorder(c).Code)
}

func TestDeclineInvite_Success(t *testing.T) {
	handler, draftStore, e := setupInvitesTest()
	userUuid := uuid.New()
	invite := model.DraftInvite{
		Id:              1,
		DraftId:         1,
		InvitedUserUuid: userUuid,
		Status:          "pending",
	}

	draftStore.On("GetInvite", mocklib.Anything, 1).Return(invite, nil)
	draftStore.On("DeclineInvite", mocklib.Anything, 1).Return(nil)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/invites/1/decline", "/api/v1/invites/:id/decline", "id", "1", nil)
	err := handler.DeclineInvite(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, responseRecorder(c).Code)
}

func TestDeclineInvite_NotFound(t *testing.T) {
	handler, draftStore, e := setupInvitesTest()
	userUuid := uuid.New()

	draftStore.On("GetInvite", mocklib.Anything, 1).Return(model.DraftInvite{}, errors.New("db error"))

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/invites/1/decline", "/api/v1/invites/:id/decline", "id", "1", nil)
	err := handler.DeclineInvite(c)

	require.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, responseRecorder(c).Code)
}

func TestDeclineInvite_WrongUser(t *testing.T) {
	handler, draftStore, e := setupInvitesTest()
	userUuid := uuid.New()
	otherUser := uuid.New()
	invite := model.DraftInvite{
		Id:              1,
		DraftId:         1,
		InvitedUserUuid: otherUser,
		Status:          "pending",
	}

	draftStore.On("GetInvite", mocklib.Anything, 1).Return(invite, nil)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/invites/1/decline", "/api/v1/invites/:id/decline", "id", "1", nil)
	err := handler.DeclineInvite(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, responseRecorder(c).Code)
}
