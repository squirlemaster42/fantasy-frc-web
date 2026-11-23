package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"server/draft"
	"server/model"
	"server/model/mocks"
	"server/picking"
	"server/utils"
)

func TestCheckWebsocketOrigin(t *testing.T) {
	tests := []struct {
		name          string
		allowedOrigin string
		host          string
		origin        string
		want          bool
	}{
		{"allowed origin exact match", "https://example.com", "example.com", "https://example.com", true},
		{"allowed origin mismatch", "https://example.com", "example.com", "https://evil.com", false},
		{"allowed origin empty rejected", "https://example.com", "example.com", "", false},
		{"same origin https", "", "example.com", "https://example.com", true},
		{"same origin http", "", "example.com", "http://example.com", true},
		{"same origin with port", "", "example.com:8080", "https://example.com:8080", true},
		{"subdomain prefix attack rejected", "", "example.com", "https://example.com.evil.com", false},
		{"different host rejected", "", "example.com", "https://other.com", false},
		{"port mismatch rejected", "", "example.com", "https://example.com:8443", false},
		{"scheme mismatch rejected", "", "example.com", "ftp://example.com", false},
		{"empty origin allowed in fallback", "", "example.com", "", true},
		{"localhost same port", "", "localhost:8080", "http://localhost:8080", true},
		{"localhost different port allowed", "", "localhost:8080", "http://localhost:3000", true},
		{"localhost hostname only", "", "localhost", "http://localhost", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Config: ConfigGroup{AllowedOrigin: tt.allowedOrigin}}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/u/draft/1/pick", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			assert.Equal(t, tt.want, h.checkWebsocketOrigin(req))
		})
	}
}

func TestServePickPage(t *testing.T) {
	t.Run("invalid draft id", func(t *testing.T) {
		_, c, _ := setupTestContext(t, http.MethodGet, "/u/draft/abc/pick", "", "test-session")
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

		err := h.ServePickPage(c)
		require.Error(t, err)
	})
}

func TestHandleSkipPickToggle(t *testing.T) {
	t.Run("success - mark skipping", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/u/draft/42/skipPickToggle", strings.NewReader("skipping=true"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"}) //nolint:gosec // test cookie
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockDraftStore.On("GetDraftPlayerId", c.Request().Context(), 42, userUuid).Return(7, nil)
		mockDraftStore.On("MarkShouldSkipPick", c.Request().Context(), 7, true).Return(nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleSkipPickToggle(c)
		require.NoError(t, err)
	})

	t.Run("invalid draft id", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/u/draft/abc/skipPickToggle", strings.NewReader("skipping=true"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
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

		err := h.HandleSkipPickToggle(c)
		require.Error(t, err)
	})

	t.Run("get draft player id fails", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/u/draft/42/skipPickToggle", strings.NewReader("skipping=true"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"}) //nolint:gosec // test cookie
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockDraftStore.On("GetDraftPlayerId", c.Request().Context(), 42, userUuid).Return(0, errors.New("db down"))

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleSkipPickToggle(c)
		require.Error(t, err)
	})
}

func setupWebsocketTestHandler(t *testing.T, mockStore *mocks.MockDraftStore) (*Handler, *picking.PickNotifier) {
	t.Helper()

	pickNotifier := &picking.PickNotifier{Watchers: make(map[int][]picking.Watcher)}
	actorMap := draft.NewDraftActorMap(mockStore, nil, nil, nil, pickNotifier, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Config: ConfigGroup{},
		Stores: StorageGroup{
			DraftStore: mockStore,
		},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}
	return h, pickNotifier
}

func startWebsocketServer(t *testing.T, h *Handler, draftId string, userUuid uuid.UUID) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := echo.New()
		c := e.NewContext(r, w)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: draftId}})
		c.Set("userUuid", userUuid)
		_ = h.PickNotifier(c)
	}))
	t.Cleanup(func() { server.Close() })
	return server
}

func dialWebsocket(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()

	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"

	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestPickNotifier_InvalidDraftId(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	h, _ := setupWebsocketTestHandler(t, mocks.NewMockDraftStore(t))
	server := startWebsocketServer(t, h, "abc", userUuid)

	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"
	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Error(t, err)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	}
	assert.Nil(t, conn)
}

func TestPickNotifier_DraftNotFound(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraftPlayerId", mock.Anything, 999, userUuid).Return(0, sql.ErrNoRows)

	h, _ := setupWebsocketTestHandler(t, mockStore)
	server := startWebsocketServer(t, h, "999", userUuid)

	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"
	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Error(t, err)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
	assert.Nil(t, conn)
}

func TestPickNotifier_ForbiddenForNonMember(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(0, sql.ErrNoRows)

	h, _ := setupWebsocketTestHandler(t, mockStore)
	server := startWebsocketServer(t, h, "1", userUuid)

	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"
	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Error(t, err)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
	assert.Nil(t, conn)
}

func TestPickNotifier_RegistersAndUnregistersWatcher(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(1, nil)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{
		Id:      1,
		Status:  model.PICKING,
		Players: []model.DraftPlayer{},
		Picks:   []model.Pick{},
	}, nil).Once()

	h, pickNotifier := setupWebsocketTestHandler(t, mockStore)
	server := startWebsocketServer(t, h, "1", userUuid)
	conn := dialWebsocket(t, server)

	// Give the handler time to register the watcher
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, pickNotifier.GetWatcherCount(1))

	// Clean disconnect
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, pickNotifier.GetWatcherCount(1))
}

func TestPickNotifier_ReceivesNotification(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(1, nil)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{
		Id:      1,
		Status:  model.PICKING,
		Players: []model.DraftPlayer{},
		Picks:   []model.Pick{},
	}, nil).Once()

	h, pickNotifier := setupWebsocketTestHandler(t, mockStore)
	server := startWebsocketServer(t, h, "1", userUuid)
	conn := dialWebsocket(t, server)

	// Wait for watcher registration
	time.Sleep(50 * time.Millisecond)

	// Trigger a notification
	pickNotifier.NotifyWatchers(t.Context(), 1)

	// Wait for the server to render and send the update
	err := conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	require.NoError(t, err)
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.NotEmpty(t, message)
}

func TestPickNotifier_ClosesWhenDraftLeavesPicking(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(1, nil)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{
		Id:      1,
		Status:  model.PICKING,
		Players: []model.DraftPlayer{},
		Picks:   []model.Pick{},
	}, nil).Once()

	h, pickNotifier := setupWebsocketTestHandler(t, mockStore)
	server := startWebsocketServer(t, h, "1", userUuid)
	conn := dialWebsocket(t, server)

	time.Sleep(50 * time.Millisecond)

	// Simulate the draft transitioning to TEAMS_PLAYING.
	pickNotifier.NotifyStateChange(t.Context(), 1, model.TEAMS_PLAYING)

	err := conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	require.NoError(t, err)
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
}

func TestPickNotifier_ErrorPaths(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("missing user uuid redirects to login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/1/pick/ws", "", "")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})

		h := &Handler{}
		err := h.PickNotifier(c)

		require.ErrorIs(t, err, errLoginRequired)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})

	t.Run("invalid draft id", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/abc/pick/ws", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})
		c.Set("userUuid", userUuid)

		h := &Handler{}
		err := h.PickNotifier(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("non-member forbidden", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(0, sql.ErrNoRows).Once()

		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/1/pick/ws", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		h := &Handler{Stores: StorageGroup{DraftStore: mockDraftStore}}
		err := h.PickNotifier(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("membership check error", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(0, assert.AnError).Once()

		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/1/pick/ws", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		h := &Handler{Stores: StorageGroup{DraftStore: mockDraftStore}}
		err := h.PickNotifier(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestPickNotifier_EnforcesReadLimit(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(1, nil)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{
		Id:      1,
		Status:  model.PICKING,
		Players: []model.DraftPlayer{},
		Picks:   []model.Pick{},
	}, nil).Once()

	h, _ := setupWebsocketTestHandler(t, mockStore)
	server := startWebsocketServer(t, h, "1", userUuid)
	conn := dialWebsocket(t, server)

	largeMessage := make([]byte, WsMaxMessageSize()+1)
	err := conn.WriteMessage(websocket.BinaryMessage, largeMessage)
	require.NoError(t, err)

	err = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	require.NoError(t, err)
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
}

func makeTestDraftModel(userUuid uuid.UUID, status model.DraftState) model.DraftModel {
	playerId := 5
	pick := model.Pick{Id: 10, Player: playerId, ExpirationTime: time.Now()}
	player := model.DraftPlayer{
		Id:          playerId,
		User:        model.User{UserUuid: userUuid},
		PlayerOrder: sql.NullInt16{Int16: 0, Valid: true},
		Picks:       []model.Pick{pick},
	}
	return model.DraftModel{
		Id:          1,
		Owner:       model.User{UserUuid: userUuid},
		Status:      status,
		CurrentPick: pick,
		NextPick:    player,
		Players:     []model.DraftPlayer{player},
		Picks:       []model.Pick{pick},
		TimingType:  model.PER_PICK,
	}
}

func TestHandlerPickRequest_EmptyPick(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 1
	playerId := 5

	draftModel := makeTestDraftModel(userUuid, model.PICKING)

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()
	mockDraftStore.On("GetDraftPlayerId", mock.Anything, draftId, userUuid).Return(playerId, nil).Once()
	mockDraftStore.On("ShouldSkipPick", mock.Anything, playerId).Return(false, nil).Once()

	body := "pickInput="
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/makePick", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Stores: StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.HandlerPickRequest(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "you must be the picking player")
}

func TestHandlerPickRequest_InvalidDraftId(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	_, c, _ := setupTestContext(t, http.MethodPost, "/u/draft/abc/makePick", "pickInput=254", "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})
	c.Set("userUuid", userUuid)

	h := &Handler{}

	err := h.HandlerPickRequest(c)
	require.Error(t, err)
}

func TestServePickPage_Success(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 1
	playerId := 5

	draftModel := makeTestDraftModel(userUuid, model.PICKING)

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()
	mockDraftStore.On("GetDraftPlayerId", mock.Anything, draftId, userUuid).Return(playerId, nil).Once()
	mockDraftStore.On("ShouldSkipPick", mock.Anything, playerId).Return(false, nil).Once()
	mockUserStore.On("GetUsername", mock.Anything, userUuid).Return("testuser", nil).Once()

	_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/1/pick", "", "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Stores: StorageGroup{DraftStore: mockDraftStore, UserStore: mockUserStore},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.ServePickPage(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandlerPickRequest_NotCurrentPick(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	otherUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	draftId := 1

	pick := model.Pick{Id: 10, Player: 5, ExpirationTime: time.Now()}
	otherPlayer := model.DraftPlayer{
		Id:          5,
		User:        model.User{UserUuid: otherUuid},
		PlayerOrder: sql.NullInt16{Int16: 0, Valid: true},
		Picks:       []model.Pick{pick},
	}
	currentPlayer := model.DraftPlayer{
		Id:          6,
		User:        model.User{UserUuid: userUuid},
		PlayerOrder: sql.NullInt16{Int16: 1, Valid: true},
		Picks:       []model.Pick{},
	}
	draftModel := model.DraftModel{
		Id:          draftId,
		Owner:       model.User{UserUuid: otherUuid},
		Status:      model.PICKING,
		CurrentPick: pick,
		NextPick:    otherPlayer,
		Players:     []model.DraftPlayer{otherPlayer, currentPlayer},
		Picks:       []model.Pick{pick},
		TimingType:  model.PER_PICK,
	}

	handlerStore := mocks.NewMockDraftStore(t)
	actorStore := mocks.NewMockDraftStore(t)
	handlerStore.On("GetDraftPlayerId", mock.Anything, draftId, userUuid).Return(6, nil).Once()
	handlerStore.On("ShouldSkipPick", mock.Anything, 6).Return(false, nil).Once()
	actorStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()

	body := "pickInput=254"
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/makePick", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Stores: StorageGroup{DraftStore: handlerStore},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.HandlerPickRequest(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "you must be the picking player")
}

func TestHandlerPickRequest_MakePickError(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 1
	playerId := 5

	draftModel := makeTestDraftModel(userUuid, model.PICKING)

	handlerStore := mocks.NewMockDraftStore(t)
	actorStore := mocks.NewMockDraftStore(t)
	handlerStore.On("GetDraftPlayerId", mock.Anything, draftId, userUuid).Return(playerId, nil).Once()
	handlerStore.On("ShouldSkipPick", mock.Anything, playerId).Return(false, nil).Once()
	actorStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()
	actorStore.On("HasBeenPicked", mock.Anything, draftId, "frc254").Return(true, nil).Once()

	body := "pickInput=254"
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/1/makePick", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
	c.Set("userUuid", userUuid)

	actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, &stubEventTeamCache{})

	h := &Handler{
		Stores: StorageGroup{DraftStore: handlerStore},
		Services: ServiceGroup{
			DraftActorMap: actorMap,
		},
	}

	err := h.HandlerPickRequest(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "team already picked")
}

func TestRenderPickPage_ErrorBranches(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftId := 1

	t.Run("actor not found", func(t *testing.T) {
		actorStore := mocks.NewMockDraftStore(t)
		actorStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{}, sql.ErrNoRows).Once()

		_, c, _ := setupTestContext(t, http.MethodGet, "/u/draft/1/pick", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(actorStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.renderPickPage(c, draftId, userUuid, "testuser", nil, true)
		require.Error(t, err)
	})

	t.Run("draft player id error", func(t *testing.T) {
		draftModel := makeTestDraftModel(userUuid, model.PICKING)

		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()
		mockDraftStore.On("GetDraftPlayerId", mock.Anything, draftId, userUuid).Return(0, assert.AnError).Once()
		mockDraftStore.On("ShouldSkipPick", mock.Anything, -1).Return(false, nil).Once()

		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/1/pick", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Stores: StorageGroup{DraftStore: mockDraftStore}, Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.renderPickPage(c, draftId, userUuid, "testuser", nil, true)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("should skip pick error", func(t *testing.T) {
		draftModel := makeTestDraftModel(userUuid, model.PICKING)

		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(draftModel, nil).Once()
		mockDraftStore.On("GetDraftPlayerId", mock.Anything, draftId, userUuid).Return(5, nil).Once()
		mockDraftStore.On("ShouldSkipPick", mock.Anything, 5).Return(false, assert.AnError).Once()

		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/1/pick", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "1"}})
		c.Set("userUuid", userUuid)

		actorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		h := &Handler{Stores: StorageGroup{DraftStore: mockDraftStore}, Services: ServiceGroup{DraftActorMap: actorMap}}

		err := h.renderPickPage(c, draftId, userUuid, "testuser", nil, true)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
