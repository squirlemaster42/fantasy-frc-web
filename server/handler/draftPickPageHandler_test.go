package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"server/draft"
	"server/model"
	"server/model/mocks"
	"server/picking"
	"server/utils"
)

func TestServePickPage(t *testing.T) {
	t.Run("invalid draft id", func(t *testing.T) {
		_, c, _ := setupTestContext(t, http.MethodGet, "/u/draft/abc/pick", "", "test-session")
		c.SetParamNames("id")
		c.SetParamValues("abc")

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
		assert.Error(t, err)
	})
}

func TestHandleSkipPickToggle(t *testing.T) {
	t.Run("success - mark skipping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/u/draft/42/skipPickToggle", strings.NewReader("skipping=true"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"})
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("42")

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
		assert.NoError(t, err)
	})

	t.Run("invalid draft id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/u/draft/abc/skipPickToggle", strings.NewReader("skipping=true"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"})
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

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
		assert.Error(t, err)
	})
}

func setupWebsocketTestHandler(t *testing.T, mockStore *mocks.MockDraftStore) (*Handler, *picking.PickNotifier) {
	t.Helper()

	pickNotifier := &picking.PickNotifier{Watchers: make(map[int][]picking.Watcher)}
	actorMap := draft.NewDraftActorMap(mockStore, nil, nil, nil, pickNotifier, utils.DefaultPickWindowConfig(), 16)

	h := &Handler{
		Config:   ConfigGroup{},
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
		c.SetParamNames("id")
		c.SetParamValues(draftId)
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
	assert.NoError(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestPickNotifier_InvalidDraftId(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	h, _ := setupWebsocketTestHandler(t, mocks.NewMockDraftStore(t))
	server := startWebsocketServer(t, h, "abc", userUuid)

	conn := dialWebsocket(t, server)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err := conn.ReadMessage()
	assert.Error(t, err)
}

func TestPickNotifier_DraftNotFound(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraft", mock.Anything, 999).Return(model.DraftModel{}, assert.AnError)

	h, _ := setupWebsocketTestHandler(t, mockStore)
	server := startWebsocketServer(t, h, "999", userUuid)

	conn := dialWebsocket(t, server)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err := conn.ReadMessage()
	assert.Error(t, err)
}

func TestPickNotifier_RegistersAndUnregistersWatcher(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
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
	assert.Len(t, pickNotifier.Watchers[1], 1)

	// Clean disconnect
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(50 * time.Millisecond)

	assert.Empty(t, pickNotifier.Watchers[1])
}

func TestPickNotifier_ReceivesNotification(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
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
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.NotEmpty(t, message)
}
