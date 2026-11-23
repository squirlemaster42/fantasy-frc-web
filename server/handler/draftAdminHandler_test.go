package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"server/background"
	"server/draft"
	"server/model"
	"server/model/mocks"
	"server/utils"
)

func TestHandleDraftAdminGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/admin", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:    42,
			Owner: model.User{UserUuid: userUuid},
		}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleDraftAdminGet(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid draft id", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/abc/admin", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleDraftAdminGet(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("draft not found redirects home", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/admin", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{}, sql.ErrNoRows)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleDraftAdminGet(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/u/home", rec.Header().Get("Location"))
	})

	t.Run("non-owner forbidden", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/admin", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		ownerUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:    42,
			Owner: model.User{UserUuid: ownerUuid},
		}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleDraftAdminGet(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandleAdminEndDraft(t *testing.T) {
	t.Run("invalid draft id", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/u/draft/abc/admin/endDraft", nil)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"}) //nolint:gosec // test cookie
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleAdminEndDraft(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("draft not found", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/u/draft/42/admin/endDraft", nil)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"}) //nolint:gosec // test cookie
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{}, sql.ErrNoRows)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleAdminEndDraft(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("non-owner forbidden", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/u/draft/42/admin/endDraft", nil)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"}) //nolint:gosec // test cookie
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		ownerUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:    42,
			Owner: model.User{UserUuid: ownerUuid},
		}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleAdminEndDraft(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleAdminSkipPick(t *testing.T) {
	t.Run("invalid draft id", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/u/draft/abc/admin/skip", nil)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"}) //nolint:gosec // test cookie
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleAdminSkipPick(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("draft not found", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/u/draft/42/admin/skip", nil)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"}) //nolint:gosec // test cookie
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{}, sql.ErrNoRows)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleAdminSkipPick(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleAdminExtendTime_Success(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 42

	draftModel := model.DraftModel{
		Id:     draftId,
		Owner:  model.User{UserUuid: userUuid},
		Status: model.PICKING,
		CurrentPick: model.Pick{
			Id:             10,
			ExpirationTime: time.Now().Add(time.Hour),
		},
	}

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Twice()
	mockDraftStore.On("UpdatePickExpirationTime", mock.Anything, 10, mock.Anything).Return(nil).Once()

	body := "duration=30m"
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/admin/extendTime", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	h := &Handler{
		Stores: StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.HandleAdminExtendTime(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Pick time extended")
}

func TestHandleAdminExtendTime_MissingDuration(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 42

	draftModel := model.DraftModel{
		Id:     draftId,
		Owner:  model.User{UserUuid: userUuid},
		Status: model.PICKING,
	}

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()

	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/admin/extendTime", "", "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	h := &Handler{
		Stores: StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.HandleAdminExtendTime(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Duration is required")
}

func TestHandleAdminExtendTime_InvalidDuration(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 42

	draftModel := model.DraftModel{
		Id:     draftId,
		Owner:  model.User{UserUuid: userUuid},
		Status: model.PICKING,
	}

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()

	body := "duration=not-a-duration"
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/admin/extendTime", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	h := &Handler{
		Stores: StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.HandleAdminExtendTime(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid duration format")
}

func TestHandleAdminExtendTime_ErrorBranches(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("non-positive duration", func(t *testing.T) {
		draftModel := model.DraftModel{
			Id:     1,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.PICKING,
		}

		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/extendTime", "duration=-5m", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Stores: StorageGroup{DraftStore: mockDraftStore}, Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.HandleAdminExtendTime(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Duration must be positive")
	})

	t.Run("actor not found", func(t *testing.T) {
		draftModel := model.DraftModel{
			Id:     1,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.PICKING,
		}

		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{}, sql.ErrNoRows).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/extendTime", "duration=30m", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Stores: StorageGroup{DraftStore: handlerStore}, Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.HandleAdminExtendTime(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Draft not found")
	})

	t.Run("modify expiration time error", func(t *testing.T) {
		draftModel := model.DraftModel{
			Id:          1,
			Owner:       model.User{UserUuid: userUuid},
			Status:      model.PICKING,
			CurrentPick: model.Pick{Id: 10, ExpirationTime: time.Now().Add(time.Hour)},
		}

		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Twice()
		mockDraftStore.On("UpdatePickExpirationTime", mock.Anything, 10, mock.Anything).Return(errors.New("db down")).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/extendTime", "duration=30m", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Stores: StorageGroup{DraftStore: mockDraftStore}, Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.HandleAdminExtendTime(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to extend time")
	})
}

func TestHandleAdminMakePick_MissingTeam(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 42

	draftModel := model.DraftModel{
		Id:     draftId,
		Owner:  model.User{UserUuid: userUuid},
		Status: model.PICKING,
	}

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()

	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/admin/makePick", "", "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	h := &Handler{
		Stores: StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.HandleAdminMakePick(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Team number is required")
}

func TestHandleAdminMakePick_NonNumericTeam(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 42

	draftModel := model.DraftModel{
		Id:     draftId,
		Owner:  model.User{UserUuid: userUuid},
		Status: model.PICKING,
	}

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()

	body := "teamNumber=abc"
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/admin/makePick", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	h := &Handler{
		Stores: StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.HandleAdminMakePick(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Team number must be numeric")
}

func TestHandleAdminUndoPick_InvalidDraftId(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/abc/admin/undoPick", "", "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})
	c.Set("userUuid", userUuid)

	h := &Handler{}

	err := h.HandleAdminUndoPick(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRenderAdminResponse_HTML(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/admin/ping", "", "test-session")

	h := &Handler{}
	err := h.renderAdminResponse(c, "Ping", "Pong", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Pong")
}

func TestRenderAdminResponse_JSON(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/admin/ping", "", "test-session")
	c.Request().Header.Set("Accept", "application/json")

	h := &Handler{}
	err := h.renderAdminResponse(c, "Ping", "Pong", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "\"success\":true")
	assert.Contains(t, rec.Body.String(), "\"message\":\"Pong\"")
}

func TestRenderAdminResponse_JSONError(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/admin/ping", "", "test-session")
	c.Request().Header.Set("Accept", "application/json")

	h := &Handler{}
	err := h.renderAdminResponse(c, "Ping", "", assert.AnError)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, strings.ToLower(rec.Body.String()), "\"success\":false")
}

type stubEventTeamCache struct{}

func (s *stubEventTeamCache) IsTeamAtEvent(_ context.Context, _ string) (bool, error) { return true, nil }
func (s *stubEventTeamCache) Refresh(_ context.Context) error                         { return nil }
func (s *stubEventTeamCache) InvalidateAndRefresh(_ context.Context) error            { return nil }
func (s *stubEventTeamCache) LastRefresh() time.Time                                  { return time.Now() }

func makeAdminDraftModel(userUuid uuid.UUID) model.DraftModel {
	playerId := 5
	pick := model.Pick{Id: 10, Player: playerId, ExpirationTime: time.Now()}
	player := model.DraftPlayer{
		Id:          playerId,
		User:        model.User{UserUuid: userUuid},
		PlayerOrder: sql.NullInt16{Int16: 0, Valid: true},
	}
	return model.DraftModel{
		Id:          1,
		Owner:       model.User{UserUuid: userUuid},
		Status:      model.PICKING,
		CurrentPick: pick,
		NextPick:    player,
		Players:     []model.DraftPlayer{player},
		TimingType:  model.PER_PICK,
	}
}

func TestHandleAdminSkipPick_Success(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 1

	initialModel := makeAdminDraftModel(userUuid)
	reloadedModel := initialModel
	reloadedModel.CurrentPick.Id = 11

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(initialModel, nil).Twice()
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(reloadedModel, nil).Once()
	mockRunInTransaction(mockDraftStore)
	mockDraftStore.On("SkipPick", mock.Anything, 10).Return(nil).Once()
	mockDraftStore.On("MakePickAvailable", mock.Anything, 5, mock.Anything, mock.Anything).Return(11, nil).Once()

	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/skip", "", "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	h := &Handler{
		Stores:   StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{DraftActorMap: actorMap},
	}

	err := h.HandleAdminSkipPick(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Pick skipped successfully")
}

func TestHandleAdminMakePick_Success(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 1

	initialModel := makeAdminDraftModel(userUuid)
	reloadedModel := initialModel
	reloadedModel.CurrentPick.Id = 11

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(initialModel, nil).Twice()
	mockDraftStore.On("HasBeenPicked", mock.Anything, draftId, "frc254").Return(false, nil).Once()
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(reloadedModel, nil).Once()
	mockRunInTransaction(mockDraftStore)
	mockDraftStore.On("MakePick", mock.Anything, mock.Anything).Return(nil).Once()
	mockDraftStore.On("MakePickAvailable", mock.Anything, 5, mock.Anything, mock.Anything).Return(11, nil).Once()

	body := "teamNumber=254"
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/makePick", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, &stubEventTeamCache{})
	h := &Handler{
		Stores:   StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{DraftActorMap: actorMap},
	}

	err := h.HandleAdminMakePick(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Successfully picked team 254")
}

func TestHandleAdminUndoPick_Success(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 1

	draftModel := makeAdminDraftModel(userUuid)
	draftModel.CurrentPick.Id = 11

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Times(3)
	mockDraftStore.On("GetPreviousPick", mock.Anything, draftId, 11).Return(model.Pick{Id: 10, Player: 5}, nil).Once()
	mockRunInTransaction(mockDraftStore)
	mockDraftStore.On("DeletePick", mock.Anything, 11).Return(nil).Once()
	mockDraftStore.On("ResetPick", mock.Anything, 10, mock.Anything).Return(nil).Once()

	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/undoPick", "", "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	h := &Handler{
		Stores:   StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{DraftActorMap: actorMap},
	}

	err := h.HandleAdminUndoPick(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Pick undone successfully")
}

func TestHandleAdminEndDraft_Success(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 1

	picks := make([]model.Pick, 8)
	for i := range picks {
		picks[i] = model.Pick{Id: i + 1, Player: 5, PickTime: sql.NullTime{Time: time.Now(), Valid: true}}
	}

	initialModel := model.DraftModel{
		Id:          draftId,
		Owner:       model.User{UserUuid: userUuid},
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: 9, Player: 5},
		Players: []model.DraftPlayer{
			{Id: 5, User: model.User{UserUuid: userUuid}, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
		Picks:      picks,
		TimingType: model.PER_PICK,
	}
	reloadedModel := initialModel
	reloadedModel.Status = model.TEAMS_PLAYING

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(initialModel, nil).Twice()
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(reloadedModel, nil).Once()
	mockRunInTransaction(mockDraftStore)
	mockDraftStore.On("SkipPick", mock.Anything, 9).Return(nil).Once()
	mockDraftStore.On("UpdateDraftStatus", mock.Anything, draftId, model.TEAMS_PLAYING).Return(nil).Once()

	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/endDraft", "", "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	h := &Handler{
		Stores:   StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{DraftActorMap: actorMap},
	}

	err := h.HandleAdminEndDraft(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Draft ended and remaining picks skipped")
}

func TestHandleAdminEndDraft_FailureBranches(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("actor not found", func(t *testing.T) {
		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		draftModel := makeAdminDraftModel(userUuid)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{}, sql.ErrNoRows).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/endDraft", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		emptyActorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: emptyActorMap},
		}

		err := h.HandleAdminEndDraft(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Draft not found")
	})

	t.Run("end draft returns false", func(t *testing.T) {
		draftModel := makeAdminDraftModel(userUuid)
		draftModel.Status = model.TEAMS_PLAYING

		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/endDraft", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: actorMap},
		}

		err := h.HandleAdminEndDraft(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to end draft")
	})

	t.Run("daemon remove error logs but succeeds", func(t *testing.T) {
		picks := make([]model.Pick, 8)
		for i := range picks {
			picks[i] = model.Pick{Id: i + 1, Player: 5, PickTime: sql.NullTime{Time: time.Now(), Valid: true}}
		}

		initialModel := model.DraftModel{
			Id:          1,
			Owner:       model.User{UserUuid: userUuid},
			Status:      model.PICKING,
			CurrentPick: model.Pick{Id: 9, Player: 5},
			Players: []model.DraftPlayer{
				{Id: 5, User: model.User{UserUuid: userUuid}, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
			},
			Picks:      picks,
			TimingType: model.PER_PICK,
		}
		reloadedModel := initialModel
		reloadedModel.Status = model.TEAMS_PLAYING

		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(initialModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(initialModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(reloadedModel, nil).Once()
		mockRunInTransaction(actorStore)
		actorStore.On("SkipPick", mock.Anything, 9).Return(nil).Once()
		actorStore.On("UpdateDraftStatus", mock.Anything, 1, model.TEAMS_PLAYING).Return(nil).Once()

		actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		daemon := background.NewDraftDaemon(actorStore, actorMap)

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/endDraft", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: actorMap, DraftDaemon: daemon},
		}

		err := h.HandleAdminEndDraft(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Draft ended and remaining picks skipped")
	})
}

func TestHandleAdminSkipPick_FailureBranches(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("actor not found", func(t *testing.T) {
		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		draftModel := makeAdminDraftModel(userUuid)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{}, sql.ErrNoRows).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/skip", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		emptyActorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: emptyActorMap},
		}

		err := h.HandleAdminSkipPick(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Draft not found")
	})

	t.Run("skip returns false", func(t *testing.T) {
		draftModel := makeAdminDraftModel(userUuid)
		draftModel.Status = model.TEAMS_PLAYING

		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/skip", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: actorMap},
		}

		err := h.HandleAdminSkipPick(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to skip pick")
	})
}

func TestHandleAdminUndoPick_FailureBranches(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("actor not found", func(t *testing.T) {
		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		draftModel := makeAdminDraftModel(userUuid)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{}, sql.ErrNoRows).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/undoPick", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		emptyActorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: emptyActorMap},
		}

		err := h.HandleAdminUndoPick(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Draft not found")
	})

	t.Run("undo returns error", func(t *testing.T) {
		draftModel := makeAdminDraftModel(userUuid)
		draftModel.CurrentPick.Id = 11

		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Maybe()
		actorStore.On("GetPreviousPick", mock.Anything, 1, 11).Return(model.Pick{}, sql.ErrNoRows).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/undoPick", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: actorMap},
		}

		err := h.HandleAdminUndoPick(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to undo pick")
	})
}

func TestHandleAdminMakePick_FailureBranches(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("actor not found", func(t *testing.T) {
		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		draftModel := makeAdminDraftModel(userUuid)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{}, sql.ErrNoRows).Once()

		body := "teamNumber=254"
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/makePick", body, "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		emptyActorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: emptyActorMap},
		}

		err := h.HandleAdminMakePick(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Draft not found")
	})

	t.Run("no current pick", func(t *testing.T) {
		draftModel := makeAdminDraftModel(userUuid)
		draftModel.CurrentPick = model.Pick{}

		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()

		body := "teamNumber=254"
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/makePick", body, "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: actorMap},
		}

		err := h.HandleAdminMakePick(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "No current pick found")
	})

	t.Run("make pick returns error", func(t *testing.T) {
		draftModel := makeAdminDraftModel(userUuid)

		handlerStore := mocks.NewMockDraftStore(t)
		actorStore := mocks.NewMockDraftStore(t)
		handlerStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("GetDraft", mock.Anything, 1).Return(draftModel, nil).Once()
		actorStore.On("HasBeenPicked", mock.Anything, 1, "frc254").Return(true, nil).Once()

		body := "teamNumber=254"
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/admin/makePick", body, "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{
			Stores:   StorageGroup{DraftStore: handlerStore},
			Services: ServiceGroup{DraftActorMap: actorMap},
		}

		err := h.HandleAdminMakePick(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to make pick")
	})
}
