package apihandler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	apimodel "server/api/model"
	"server/model"
	"server/model/mocks"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	mocklib "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"uuid"
)

func setupDraftsTest() (*Handler, *mocks.MockDraftStore, *mocks.MockUserStore, *mocks.MockTeamStore, *echo.Echo) {
	draftStore := new(mocks.MockDraftStore)
	userStore := new(mocks.MockUserStore)
	teamStore := new(mocks.MockTeamStore)
	handler := NewHandler(draftStore, userStore, nil, teamStore, nil)
	e := echo.New()
	return handler, draftStore, userStore, teamStore, e
}

func newDraftContext(ctx context.Context, e *echo.Echo, userUuid uuid.UUID, method, path string, body []byte) *echo.Context {
	req := httptest.NewRequestWithContext(ctx, method, path, bytes.NewReader(body))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userUuid", userUuid)
	return c
}

func newDraftContextWithParam(ctx context.Context, e *echo.Echo, userUuid uuid.UUID, method, path, routePath, paramName, paramValue string, body []byte) *echo.Context {
	c := newDraftContext(ctx, e, userUuid, method, path, body)
	ri := echo.RouteInfo{Method: method, Path: routePath, Parameters: []string{paramName}}
	pv := echo.PathValues{{Name: paramName, Value: paramValue}}
	c.InitializeRoute(&ri, &pv)
	return c
}

func TestListDrafts_Success(t *testing.T) {
	handler, draftStore, _, _, e := setupDraftsTest()
	userUuid := uuid.New()
	drafts := []model.DraftModel{
		{
			Id:          1,
			DisplayName: "Draft One",
			Status:      model.FILLING,
			Owner:       model.User{UserUuid: userUuid, Username: "owner"},
		},
	}

	draftStore.On("GetDraftsForUser", mocklib.Anything, userUuid).Return(drafts, nil)

	c := newDraftContext(t.Context(), e, userUuid, http.MethodGet, "/api/v1/drafts", nil)
	err := handler.ListDrafts(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, responseRecorder(c).Code)

	var resp []apimodel.DraftResponse
	require.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	assert.Equal(t, 1, resp[0].Id)
	assert.Equal(t, "Draft One", resp[0].DisplayName)
}

func TestListDrafts_StoreError(t *testing.T) {
	handler, draftStore, _, _, e := setupDraftsTest()
	userUuid := uuid.New()

	draftStore.On("GetDraftsForUser", mocklib.Anything, userUuid).Return(nil, errors.New("db error"))

	c := newDraftContext(t.Context(), e, userUuid, http.MethodGet, "/api/v1/drafts", nil)
	err := handler.ListDrafts(c)

	require.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, responseRecorder(c).Code)
}

func TestGetDraft_Success(t *testing.T) {
	handler, draftStore, _, _, e := setupDraftsTest()
	userUuid := uuid.New()
	draft := model.DraftModel{
		Id:          1,
		DisplayName: "Draft One",
		Status:      model.FILLING,
		Owner:       model.User{UserUuid: userUuid, Username: "owner"},
		Players: []model.DraftPlayer{
			{User: model.User{UserUuid: userUuid, Username: "player"}},
		},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodGet, "/api/v1/drafts/1", "/api/v1/drafts/:id", "id", "1", nil)
	err := handler.GetDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, responseRecorder(c).Code)

	var resp apimodel.DraftResponse
	require.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Id)
	assert.Equal(t, "Draft One", resp.DisplayName)
}

func TestGetDraft_NotFound(t *testing.T) {
	handler, draftStore, _, _, e := setupDraftsTest()
	userUuid := uuid.New()

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(model.DraftModel{}, sql.ErrNoRows)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodGet, "/api/v1/drafts/1", "/api/v1/drafts/:id", "id", "1", nil)
	err := handler.GetDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, responseRecorder(c).Code)
}

func TestGetDraft_NotAMember(t *testing.T) {
	handler, draftStore, _, _, e := setupDraftsTest()
	userUuid := uuid.New()
	otherUser := uuid.New()
	draft := model.DraftModel{
		Id:          1,
		DisplayName: "Draft One",
		Status:      model.FILLING,
		Owner:       model.User{UserUuid: otherUser, Username: "owner"},
		Players: []model.DraftPlayer{
			{User: model.User{UserUuid: otherUser, Username: "player"}},
		},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodGet, "/api/v1/drafts/1", "/api/v1/drafts/:id", "id", "1", nil)
	err := handler.GetDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, responseRecorder(c).Code)
}

func TestCreateDraft_Success(t *testing.T) {
	handler, draftStore, _, _, e := setupDraftsTest()
	userUuid := uuid.New()

	draftStore.On("CreateDraft", mocklib.Anything, mocklib.AnythingOfType("*model.DraftModel")).Return(1, nil)

	reqBody, err := json.Marshal(apimodel.CreateDraftRequest{
		DisplayName:       "New Draft",
		TimingType:        "increment",
		IncrementTimeSec:  60,
		PerPickExpTimeSec: 300,
	})
	require.NoError(t, err)

	c := newDraftContext(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts", reqBody)
	err = handler.CreateDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, responseRecorder(c).Code)

	var resp apimodel.DraftResponse
	require.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Id)
	assert.Equal(t, "New Draft", resp.DisplayName)
	assert.Equal(t, "increment", resp.TimingType)
}

func TestCreateDraft_MissingDisplayName(t *testing.T) {
	handler, _, _, _, e := setupDraftsTest()
	userUuid := uuid.New()

	reqBody, err := json.Marshal(apimodel.CreateDraftRequest{
		TimingType:        "increment",
		IncrementTimeSec:  60,
		PerPickExpTimeSec: 300,
	})
	require.NoError(t, err)

	c := newDraftContext(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts", reqBody)
	err = handler.CreateDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, responseRecorder(c).Code)
}

func TestCreateDraft_InvalidTimingType(t *testing.T) {
	handler, _, _, _, e := setupDraftsTest()
	userUuid := uuid.New()

	reqBody, err := json.Marshal(apimodel.CreateDraftRequest{
		DisplayName:       "New Draft",
		TimingType:        "invalid",
		IncrementTimeSec:  60,
		PerPickExpTimeSec: 300,
	})
	require.NoError(t, err)

	c := newDraftContext(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts", reqBody)
	err = handler.CreateDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, responseRecorder(c).Code)
}

func TestGetDraftScore_Success(t *testing.T) {
	handler, draftStore, _, _, e := setupDraftsTest()
	userUuid := uuid.New()
	draft := model.DraftModel{
		Id:          1,
		DisplayName: "Draft One",
		Status:      model.PICKING,
		Owner:       model.User{UserUuid: userUuid, Username: "owner"},
		Players: []model.DraftPlayer{
			{User: model.User{UserUuid: userUuid, Username: "player"}},
		},
	}
	players := []model.DraftPlayer{
		{User: model.User{UserUuid: userUuid, Username: "player"}, Score: 10},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)
	draftStore.On("GetDraftScore", mocklib.Anything, 1).Return(players, nil)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodGet, "/api/v1/drafts/1/score", "/api/v1/drafts/:id/score", "id", "1", nil)
	err := handler.GetDraftScore(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, responseRecorder(c).Code)

	var resp apimodel.DraftScoreResponse
	require.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.DraftId)
	assert.Len(t, resp.Players, 1)
	assert.Equal(t, 10, resp.Players[0].Score)
}

func TestGetTeamScore_Success(t *testing.T) {
	handler, _, _, teamStore, e := setupDraftsTest()

	teamStore.On("GetScore", mocklib.Anything, "frc254").Return(map[string]int{"qual": 10}, nil)
	teamStore.On("GetMatchScores", mocklib.Anything, "frc254").Return([]model.MatchTeamScore{
		{MatchTbaId: "2024abc_qm1", Alliance: "red", Score: 10, IsDqed: false},
	}, nil)

	c := newDraftContext(t.Context(), e, uuid.Nil(), http.MethodGet, "/api/v1/team/score?team_number=254", nil)
	err := handler.GetTeamScore(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, responseRecorder(c).Code)

	var resp apimodel.TeamScoreResponse
	require.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	assert.Equal(t, 254, resp.TeamNumber)
	assert.Equal(t, 10, resp.Scores["qual"])
	assert.Len(t, resp.Matches, 1)
}

func TestGetTeamScore_MissingTeamNumber(t *testing.T) {
	handler, _, _, _, e := setupDraftsTest()

	c := newDraftContext(t.Context(), e, uuid.Nil(), http.MethodGet, "/api/v1/team/score", nil)
	err := handler.GetTeamScore(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, responseRecorder(c).Code)
}

func TestSearchUsers_Success(t *testing.T) {
	handler, draftStore, userStore, _, e := setupDraftsTest()
	userUuid := uuid.New()
	draft := model.DraftModel{
		Id:    1,
		Owner: model.User{UserUuid: userUuid, Username: "owner"},
	}
	users := []model.User{
		{UserUuid: uuid.New(), Username: "founduser"},
	}

	draftStore.On("GetDraft", mocklib.Anything, 1).Return(draft, nil)
	userStore.On("SearchUsers", mocklib.Anything, "found", 1).Return(users, nil)

	c := newDraftContext(t.Context(), e, userUuid, http.MethodGet, "/api/v1/users/search?q=found&draft_id=1", nil)
	err := handler.SearchUsers(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, responseRecorder(c).Code)

	var resp apimodel.UserSearchResponse
	require.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	require.Len(t, resp.Users, 1)
	assert.Equal(t, "founduser", resp.Users[0].Username)
}

func TestSearchUsers_MissingParams(t *testing.T) {
	handler, _, _, _, e := setupDraftsTest()
	userUuid := uuid.New()

	c := newDraftContext(t.Context(), e, userUuid, http.MethodGet, "/api/v1/users/search", nil)
	err := handler.SearchUsers(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, responseRecorder(c).Code)
}

func TestGetLeaderboard_Success(t *testing.T) {
	handler, draftStore, _, _, e := setupDraftsTest()

	leaderboard := model.LeaderboardPage{
		Entries: []model.LeaderboardEntry{
			{
				User:      model.User{UserUuid: uuid.New(), Username: "leader"},
				Score:     100,
				DraftId:   1,
				DraftName: "Draft One",
			},
		},
		CurrentPage: 0,
		TotalPages:  1,
		PerPage:     25,
		Total:       1,
	}

	draftStore.On("GetOverallLeaderboard", mocklib.Anything, 0, 25).Return(leaderboard, nil)

	c := newDraftContext(t.Context(), e, uuid.Nil(), http.MethodGet, "/api/v1/leaderboard", nil)
	err := handler.GetLeaderboard(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, responseRecorder(c).Code)

	var resp apimodel.LeaderboardResponse
	require.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	require.Len(t, resp.Entries, 1)
	assert.Equal(t, 100, resp.Entries[0].Score)
	assert.Equal(t, "leader", resp.Entries[0].User.Username)
}

func TestToggleSkip_Success(t *testing.T) {
	handler, draftStore, _, _, e := setupDraftsTest()
	userUuid := uuid.New()

	draftStore.On("GetDraftPlayerId", mocklib.Anything, 1, userUuid).Return(42, nil)
	draftStore.On("MarkShouldSkipPick", mocklib.Anything, 42, true).Return(nil)

	reqBody, err := json.Marshal(apimodel.SkipPickRequest{Skipping: true})
	require.NoError(t, err)

	c := newDraftContextWithParam(t.Context(), e, userUuid, http.MethodPost, "/api/v1/drafts/1/skip", "/api/v1/drafts/:id/skip", "id", "1", reqBody)
	err = handler.ToggleSkip(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, responseRecorder(c).Code)
}
