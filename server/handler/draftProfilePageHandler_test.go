package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"

	"server/background"
	"server/draft"
	"server/model"
	"server/model/mocks"
	"server/utils"
)

func TestHandleViewDraftProfile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(
			t,
			http.MethodGet,
			"/u/draft/42/profile",
			"",
			"test-session",
		)

		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.
			On("GetUsername", c.Request().Context(), userUuid).
			Return("testuser", nil)

		mockDraftStore.
			On("GetDraft", c.Request().Context(), 42).
			Return(model.DraftModel{
				Id:     42,
				Owner:  model.User{UserUuid: userUuid},
				Status: model.FILLING,
			}, nil)

		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
			Services: ServiceGroup{
				DraftActorMap: draftActorMap,
			},
		}

		err := h.HandleViewDraftProfile(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid draft id", func(t *testing.T) {
		_, c, rec := setupTestContext(
			t,
			http.MethodGet,
			"/u/draft/abc/profile",
			"",
			"test-session",
		)

		c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.
			On("GetUsername", c.Request().Context(), userUuid).
			Return("testuser", nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleViewDraftProfile(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid draft ID")
	})

	t.Run("draft not found redirects home", func(t *testing.T) {
		_, c, rec := setupTestContext(
			t,
			http.MethodGet,
			"/u/draft/42/profile",
			"",
			"test-session",
		)

		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.
			On("GetUsername", c.Request().Context(), userUuid).
			Return("testuser", nil)

		mockDraftStore.
			On("GetDraft", c.Request().Context(), 42).
			Return(model.DraftModel{}, sql.ErrNoRows)

		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
			Services: ServiceGroup{
				DraftActorMap: draftActorMap,
			},
		}

		err := h.HandleViewDraftProfile(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/u/home", rec.Header().Get("Location"))
	})
}

func TestSearchPlayers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), 
			http.MethodPost,
			"/u/draft/42/searchPlayers",
			strings.NewReader("search=john"),
		)

		req.Header.Set(
			echo.HeaderContentType,
			echo.MIMEApplicationForm,
		)

		req.Header.Set(
			"Hx-Current-Url",
			"http://localhost/u/draft/42/profile",
		)

		req.AddCookie(&http.Cookie{ //nolint:gosec // test cookie
			Name:  "sessionToken",
			Value: "test-session",
		})

		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockDraftStore.
			On("GetDraft", c.Request().Context(), 42).
			Return(model.DraftModel{
				Id:    42,
				Owner: model.User{UserUuid: userUuid},
			}, nil)

		mockUserStore.
			On("SearchUsers", c.Request().Context(), "john", 42).
			Return([]model.User{
				{
					UserUuid: uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
					Username: "john_doe",
				},
			}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.SearchPlayers(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing header", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/searchPlayers", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		h := &Handler{}
		err := h.SearchPlayers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid draft id in url", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/abc/searchPlayers", "", "test-session")
		c.Request().Header.Set("Hx-Current-Url", "/u/draft/abc/profile")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		h := &Handler{}
		err := h.SearchPlayers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("search error", func(t *testing.T) {
		mockUserStore := mocks.NewMockUserStore(t)
		mockUserStore.On("SearchUsers", mock.Anything, "", 42).Return(nil, errors.New("db down")).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/searchPlayers", "", "test-session")
		c.Request().Header.Set("Hx-Current-Url", "/u/draft/42/profile")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		h := &Handler{Stores: StorageGroup{UserStore: mockUserStore}}
		err := h.SearchPlayers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("draft not found", func(t *testing.T) {
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockUserStore.On("SearchUsers", mock.Anything, "", 42).Return([]model.User{}, nil).Once()
		mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{}, sql.ErrNoRows).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/searchPlayers", "", "test-session")
		c.Request().Header.Set("Hx-Current-Url", "/u/draft/42/profile")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		h := &Handler{Stores: StorageGroup{UserStore: mockUserStore, DraftStore: mockDraftStore}}
		err := h.SearchPlayers(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestInviteDraftPlayer_Success(t *testing.T) {
	ownerUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	invitedUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	draftId := 42

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockUserStore := mocks.NewMockUserStore(t)

	draftModel := model.DraftModel{
		Id:      draftId,
		Owner:   model.User{UserUuid: ownerUuid},
		Status:  model.FILLING,
		Players: []model.DraftPlayer{},
	}

	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()
	mockDraftStore.On("InvitePlayer", mock.Anything, draftId, ownerUuid, invitedUuid).Return(1, nil).Once()
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()
	mockUserStore.On("SearchUsers", mock.Anything, "", draftId).Return([]model.User{}, nil).Once()

	body := "userUuid=" + invitedUuid.String() + "&search="
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/invite", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	c.Set("userUuid", ownerUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.InviteDraftPlayer(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestInviteDraftPlayer_NotOwner(t *testing.T) {
	ownerUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	otherUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	invitedUuid := uuid.MustParse("770e8400-e29b-41d4-a716-446655440002")
	draftId := 42

	mockDraftStore := mocks.NewMockDraftStore(t)

	draftModel := model.DraftModel{
		Id:      draftId,
		Owner:   model.User{UserUuid: ownerUuid},
		Status:  model.FILLING,
		Players: []model.DraftPlayer{},
	}

	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()

	body := "userUuid=" + invitedUuid.String() + "&search="
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/invite", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	c.Set("userUuid", otherUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
		},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.InviteDraftPlayer(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "must own the draft")
}

func TestHandleUninvitePlayer_Success(t *testing.T) {
	ownerUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 42
	inviteId := 5

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockUserStore := mocks.NewMockUserStore(t)

	draftModel := model.DraftModel{
		Id:      draftId,
		Owner:   model.User{UserUuid: ownerUuid},
		Status:  model.FILLING,
		Players: []model.DraftPlayer{},
	}

	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()
	mockDraftStore.On("UninvitePlayer", mock.Anything, draftId, ownerUuid, inviteId).Return(nil).Once()
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()
	mockUserStore.On("SearchUsers", mock.Anything, "", draftId).Return([]model.User{}, nil).Once()

	body := "inviteId=5&search="
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/uninvite", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	c.Set("userUuid", ownerUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.HandleUninvitePlayer(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleUninvitePlayer_InvalidInviteId(t *testing.T) {
	ownerUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockDraftStore := mocks.NewMockDraftStore(t)

	body := "inviteId=abc&search="
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/uninvite", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	c.Set("userUuid", ownerUuid)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
		},
	}

	err := h.HandleUninvitePlayer(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid invite ID")
}

func TestHandleStartDraft_AddsDraftToDaemon(t *testing.T) {
	_, c, rec := setupTestContext(
		t,
		http.MethodPost,
		"/u/draft/1/startDraft",
		"",
		"test-session",
	)

	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockDraftStore := mocks.NewMockDraftStore(t)

	players := make([]model.DraftPlayer, 8)
	for i := range players {
		players[i] = model.DraftPlayer{Pending: false}
	}

	mockDraftStore.
		On("GetDraft", c.Request().Context(), 1).
		Return(model.DraftModel{
			Id:      1,
			Owner:   model.User{UserUuid: userUuid},
			Status:  model.FILLING,
			Players: players,
		}, nil).Once()
	mockDraftStore.On("CancelOutstandingInvites", c.Request().Context(), 1).Return(nil).Maybe()
	mockRunInTransaction(mockDraftStore)
	mockDraftStore.On("RandomizePickOrder", c.Request().Context(), 1).Return(nil).Once()
	mockDraftStore.On("NextPick", c.Request().Context(), 1).Return(model.DraftPlayer{Id: 1}, nil).Once()
	mockDraftStore.On("MakePickAvailable", c.Request().Context(), 1, mock.Anything, mock.Anything).Return(1, nil).Once()
	mockDraftStore.On("UpdateDraftStatus", c.Request().Context(), 1, model.PICKING).Return(nil).Once()
	mockDraftStore.On("GetDraft", c.Request().Context(), 1).Return(model.DraftModel{
		Id:      1,
		Owner:   model.User{UserUuid: userUuid},
		Status:  model.PICKING,
		Players: players,
	}, nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	draftDaemon := background.NewDraftDaemon(mockDraftStore, draftActorMap)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
		},
		Services: ServiceGroup{
			DraftActorMap: draftActorMap,
			DraftDaemon:   draftDaemon,
		},
	}

	err := h.HandleStartDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/u/draft/1/profile", rec.Header().Get("HX-Redirect"))

	err = draftDaemon.AddDraft(c.Request().Context(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already added")
	mockDraftStore.AssertExpectations(t)
}

func TestHandleStartDraft_InvalidDraftId(t *testing.T) {
	_, c, rec := setupTestContext(
		t,
		http.MethodPost,
		"/u/draft/abc/startDraft",
		"",
		"test-session",
	)

	c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	h := &Handler{}

	err := h.HandleStartDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Draft Id is not a number")
}

func TestHandleStartDraft_DraftLoadError(t *testing.T) {
	_, c, rec := setupTestContext(
		t,
		http.MethodPost,
		"/u/draft/1/startDraft",
		"",
		"test-session",
	)

	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.
		On("GetDraft", c.Request().Context(), 1).
		Return(model.DraftModel{}, errors.New("draft not found"))

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
		},
		Services: ServiceGroup{
			DraftActorMap: draftActorMap,
		},
	}

	err := h.HandleStartDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Could not load draft")
}

func TestHandleStartDraft_NonOwner(t *testing.T) {
	_, c, rec := setupTestContext(
		t,
		http.MethodPost,
		"/u/draft/1/startDraft",
		"",
		"test-session",
	)

	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	ownerUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	c.Set("userUuid", userUuid)

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.
		On("GetDraft", c.Request().Context(), 1).
		Return(model.DraftModel{
			Id:     1,
			Owner:  model.User{UserUuid: ownerUuid},
			Status: model.FILLING,
			Players: []model.DraftPlayer{
				{Pending: false},
				{Pending: false},
			},
		}, nil)

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
		},
		Services: ServiceGroup{
			DraftActorMap: draftActorMap,
		},
	}

	err := h.HandleStartDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Permission Denied")
}

func TestHandleStartDraft_NotStartable(t *testing.T) {
	_, c, rec := setupTestContext(
		t,
		http.MethodPost,
		"/u/draft/1/startDraft",
		"",
		"test-session",
	)

	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.
		On("GetDraft", c.Request().Context(), 1).
		Return(model.DraftModel{
			Id:     1,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.FILLING,
			Players: []model.DraftPlayer{
				{Pending: false},
			},
		}, nil)

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
		},
		Services: ServiceGroup{
			DraftActorMap: draftActorMap,
		},
	}

	err := h.HandleStartDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Draft must have between")
}

func TestHandleUpdateDraftProfile(t *testing.T) {
	t.Run("success updates timing fields", func(t *testing.T) {
		body := "draftName=Updated&description=Updated+desc&timingType=increment&initialPickTimeSec=3600&incrementTimeSec=900&perPickExpTimeSec=7200"
		_, c, rec := setupTestContext(
			t,
			http.MethodPost,
			"/u/draft/1/profile",
			body,
			"test-session",
		)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		mockDraftStore := mocks.NewMockDraftStore(t)
		draftModel := model.DraftModel{
			Id:     1,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.FILLING,
		}
		mockDraftStore.
			On("GetDraft", c.Request().Context(), 1).
			Return(draftModel, nil).Once()
		mockDraftStore.
			On("GetDraft", c.Request().Context(), 1).
			Return(draftModel, nil).Once()
		mockDraftStore.
			On("UpdateDraft", c.Request().Context(), mock.MatchedBy(func(d *model.DraftModel) bool {
				return d.Id == 1 &&
					d.DisplayName == "Updated" &&
					d.Description == "Updated desc" &&
					d.TimingType == model.INCREMENT &&
					d.InitialPickTimeSec == 3600 &&
					d.IncrementTimeSec == 900 &&
					d.PerPickExpTimeSec == 7200
			})).
			Return(nil).Once()

		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
			},
			Services: ServiceGroup{
				DraftActorMap: draftActorMap,
			},
		}

		err := h.HandleUpdateDraftProfile(c)

		require.NoError(t, err)
		assert.Equal(t, "/u/draft/1/profile", rec.Header().Get("HX-Redirect"))
		mockDraftStore.AssertExpectations(t)
	})

	t.Run("rejects update when draft not filling", func(t *testing.T) {
		_, c, rec := setupTestContext(
			t,
			http.MethodPost,
			"/u/draft/1/profile",
			"draftName=Updated",
			"test-session",
		)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.
			On("GetDraft", c.Request().Context(), 1).
			Return(model.DraftModel{
				Id:     1,
				Owner:  model.User{UserUuid: userUuid},
				Status: model.PICKING,
			}, nil).Once()

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
			},
		}

		err := h.HandleUpdateDraftProfile(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Draft settings can only be changed while the draft is filling")
	})

	t.Run("rejects invalid Discord webhook URL", func(t *testing.T) {
		_, c, rec := setupTestContext(
			t,
			http.MethodPost,
			"/u/draft/1/profile",
			"draftName=Updated&discordWebhook=https://evil.com/webhook",
			"test-session",
		)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.
			On("GetDraft", c.Request().Context(), 1).
			Return(model.DraftModel{
				Id:     1,
				Owner:  model.User{UserUuid: userUuid},
				Status: model.FILLING,
			}, nil).Once()

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
			},
		}

		err := h.HandleUpdateDraftProfile(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Discord")
	})

	t.Run("rejects invalid timing values", func(t *testing.T) {
		body := "draftName=Updated&timingType=increment&initialPickTimeSec=not-a-number"
		_, c, rec := setupTestContext(
			t,
			http.MethodPost,
			"/u/draft/1/profile",
			body,
			"test-session",
		)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.
			On("GetDraft", c.Request().Context(), 1).
			Return(model.DraftModel{
				Id:     1,
				Owner:  model.User{UserUuid: userUuid},
				Status: model.FILLING,
			}, nil).Once()

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
			},
		}

		err := h.HandleUpdateDraftProfile(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid value for initialPickTimeSec")
	})

	t.Run("returns error when update fails", func(t *testing.T) {
		_, c, _ := setupTestContext(
			t,
			http.MethodPost,
			"/u/draft/1/profile",
			"draftName=Updated",
			"test-session",
		)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)

		mockDraftStore := mocks.NewMockDraftStore(t)
		draftModel := model.DraftModel{
			Id:     1,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.FILLING,
		}
		mockDraftStore.
			On("GetDraft", c.Request().Context(), 1).
			Return(draftModel, nil).Once()
		mockDraftStore.
			On("GetDraft", c.Request().Context(), 1).
			Return(draftModel, nil).Once()
		mockDraftStore.
			On("UpdateDraft", c.Request().Context(), mock.Anything).
			Return(errors.New("db down")).Once()

		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
			},
			Services: ServiceGroup{
				DraftActorMap: draftActorMap,
			},
		}

		err := h.HandleUpdateDraftProfile(c)

		require.Error(t, err)
	})
}

func TestInviteDraftPlayer_ErrorPaths(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	otherUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")

	t.Run("invalid user uuid", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/invite", "userUuid=not-a-uuid&search=", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		c.Set("userUuid", userUuid)

		h := &Handler{}
		err := h.InviteDraftPlayer(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid user UUID")
	})

	t.Run("draft not filling", func(t *testing.T) {
		invitedUuid := uuid.MustParse("770e8400-e29b-41d4-a716-446655440002")
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{
			Id:     42,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.PICKING,
		}, nil).Once()

		body := "userUuid=" + invitedUuid.String() + "&search="
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/invite", body, "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.InviteDraftPlayer(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "FILLING state")
	})

	t.Run("not owner", func(t *testing.T) {
		invitedUuid := uuid.MustParse("770e8400-e29b-41d4-a716-446655440002")
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{
			Id:     42,
			Owner:  model.User{UserUuid: otherUuid},
			Status: model.FILLING,
		}, nil).Once()

		body := "userUuid=" + invitedUuid.String() + "&search="
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/invite", body, "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.InviteDraftPlayer(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "must own the draft")
	})

	t.Run("invite store error", func(t *testing.T) {
		invitedUuid := uuid.MustParse("770e8400-e29b-41d4-a716-446655440002")
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{
			Id:     42,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.FILLING,
		}, nil).Once()
		mockDraftStore.On("InvitePlayer", mock.Anything, 42, userUuid, invitedUuid).Return(0, errors.New("db down")).Once()

		body := "userUuid=" + invitedUuid.String() + "&search="
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/invite", body, "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.InviteDraftPlayer(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to invite player")
	})
}

func TestHandleUninvitePlayer_ErrorPaths(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	otherUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")

	t.Run("draft not filling", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{
			Id:     42,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.PICKING,
		}, nil).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/uninvite", "inviteId=5&search=", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.HandleUninvitePlayer(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "FILLING state")
	})

	t.Run("not owner", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{
			Id:     42,
			Owner:  model.User{UserUuid: otherUuid},
			Status: model.FILLING,
		}, nil).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/uninvite", "inviteId=5&search=", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.HandleUninvitePlayer(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "must own the draft")
	})

	t.Run("uninvite store error", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{
			Id:     42,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.FILLING,
		}, nil).Once()
		mockDraftStore.On("UninvitePlayer", mock.Anything, 42, userUuid, 5).Return(errors.New("db down")).Once()

		_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/uninvite", "inviteId=5&search=", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.HandleUninvitePlayer(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to uninvite player")
	})
}
