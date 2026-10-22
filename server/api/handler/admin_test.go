package apihandler

import (
	"encoding/json"
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

func setupAdminTest() (*Handler, *mocks.MockDraftStore, *echo.Echo) {
	draftStore := new(mocks.MockDraftStore)
	handler := NewHandler(draftStore, nil, nil, nil, nil)
	e := echo.New()
	return handler, draftStore, e
}

func TestAdminExtendTime_NotOwner(t *testing.T) {
	handler, draftStore, e := setupAdminTest()
	userUuid := uuid.New()
	ownerUuid := uuid.New()
	draft := model.DraftModel{
		Id:    1,
		Owner: model.User{UserUuid: ownerUuid, Username: "owner"},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)

	reqBody, err := json.Marshal(apimodel.AdminExtendTimeRequest{Duration: apimodel.DurationString(30 * time.Minute)})
	require.NoError(t, err)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts/1/admin/extend-time", "/api/v1/drafts/:id/admin/extend-time", "id", "1", reqBody)
	err = handler.AdminExtendTime(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, responseRecorder(c).Code)
}

func TestAdminExtendTime_MissingDuration(t *testing.T) {
	handler, draftStore, e := setupAdminTest()
	userUuid := uuid.New()
	draft := model.DraftModel{
		Id:    1,
		Owner: model.User{UserUuid: userUuid, Username: "owner"},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)

	reqBody, err := json.Marshal(apimodel.AdminExtendTimeRequest{})
	require.NoError(t, err)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts/1/admin/extend-time", "/api/v1/drafts/:id/admin/extend-time", "id", "1", reqBody)
	err = handler.AdminExtendTime(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, responseRecorder(c).Code)
}

func TestAdminMakePick_NotOwner(t *testing.T) {
	handler, draftStore, e := setupAdminTest()
	userUuid := uuid.New()
	ownerUuid := uuid.New()
	draft := model.DraftModel{
		Id:    1,
		Owner: model.User{UserUuid: ownerUuid, Username: "owner"},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)

	reqBody, err := json.Marshal(apimodel.AdminMakePickRequest{TeamNumber: 254})
	require.NoError(t, err)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts/1/admin/make-pick", "/api/v1/drafts/:id/admin/make-pick", "id", "1", reqBody)
	err = handler.AdminMakePick(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, responseRecorder(c).Code)
}

func TestAdminMakePick_InvalidTeamNumber(t *testing.T) {
	handler, draftStore, e := setupAdminTest()
	userUuid := uuid.New()
	draft := model.DraftModel{
		Id:    1,
		Owner: model.User{UserUuid: userUuid, Username: "owner"},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)

	reqBody, err := json.Marshal(apimodel.AdminMakePickRequest{TeamNumber: 0})
	require.NoError(t, err)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts/1/admin/make-pick", "/api/v1/drafts/:id/admin/make-pick", "id", "1", reqBody)
	err = handler.AdminMakePick(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, responseRecorder(c).Code)
}

func TestAdminUndoPick_NotOwner(t *testing.T) {
	handler, draftStore, e := setupAdminTest()
	userUuid := uuid.New()
	ownerUuid := uuid.New()
	draft := model.DraftModel{
		Id:    1,
		Owner: model.User{UserUuid: ownerUuid, Username: "owner"},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts/1/admin/undo-pick", "/api/v1/drafts/:id/admin/undo-pick", "id", "1", nil)
	err := handler.AdminUndoPick(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, responseRecorder(c).Code)
}

func TestAdminSkipPick_NotOwner(t *testing.T) {
	handler, draftStore, e := setupAdminTest()
	userUuid := uuid.New()
	ownerUuid := uuid.New()
	draft := model.DraftModel{
		Id:    1,
		Owner: model.User{UserUuid: ownerUuid, Username: "owner"},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts/1/admin/skip-pick", "/api/v1/drafts/:id/admin/skip-pick", "id", "1", nil)
	err := handler.AdminSkipPick(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, responseRecorder(c).Code)
}
