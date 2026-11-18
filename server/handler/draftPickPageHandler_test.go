package handler

import (
	"database/sql"
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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"

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
