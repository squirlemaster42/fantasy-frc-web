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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"server/cache"
	"server/draft"
	"server/model"
	"server/model/mocks"
	"server/picking"
	"server/utils"
)

func TestHandleDraftScore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/draftScore", "", "test-session")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:     42,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.PICKING,
		}, nil)
		mockDraftStore.On("GetDraftScore", c.Request().Context(), 42).Return([]model.DraftPlayer{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
			Services: ServiceGroup{
				AvatarStore: &mockAvatarStore{color: cache.DefaultAvatarColor},
			},
		}

		err := h.HandleDraftScore(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleDraftTeamScore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/team/254", "", "test-session")
		c.SetPathValues(echo.PathValues{
			{Name: "id", Value: "42"},
			{Name: "teamNumber", Value: "254"},
		})
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockTeamStore := mocks.NewMockTeamStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:     42,
			Owner:  model.User{UserUuid: userUuid},
			Status: model.PICKING,
		}, nil)
		mockTeamStore.On("GetScore", c.Request().Context(), "frc254").Return(map[string]int{"total": 42}, nil)
		mockTeamStore.On("GetMatchScores", c.Request().Context(), "frc254").Return([]model.MatchTeamScore{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
				TeamStore:  mockTeamStore,
			},
			Services: ServiceGroup{
				AvatarStore: &mockAvatarStore{color: cache.DefaultAvatarColor},
			},
		}

		err := h.HandleDraftTeamScore(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func setupScoreWebsocketTestHandler(t *testing.T, mockStore *mocks.MockDraftStore) (*Handler, *picking.PickNotifier) {
	t.Helper()

	pickNotifier := &picking.PickNotifier{Watchers: make(map[int][]picking.Watcher)}
	actorMap := draft.NewDraftActorMap(mockStore, nil, nil, nil, pickNotifier, utils.DefaultPickWindowConfig(), 16, nil)

	h := &Handler{
		Config:   ConfigGroup{},
		Stores:   StorageGroup{DraftStore: mockStore},
		Services: ServiceGroup{DraftActorMap: actorMap, AvatarStore: &mockAvatarStore{color: cache.DefaultAvatarColor}},
	}
	return h, pickNotifier
}

func startScoreWebsocketServer(t *testing.T, h *Handler, draftId string, userUuid uuid.UUID) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := echo.New()
		c := e.NewContext(r, w)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: draftId}})
		c.Set("userUuid", userUuid)
		_ = h.ScoreNotifier(c)
	}))
	t.Cleanup(func() { server.Close() })
	return server
}

func dialScoreWebsocket(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()

	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"

	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err, "failed to dial score websocket")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestScoreNotifier_RejectNotTeamsPlaying(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(1, nil)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{
		Id:     1,
		Status: model.PICKING,
	}, nil)

	h, _ := setupScoreWebsocketTestHandler(t, mockStore)
	server := startScoreWebsocketServer(t, h, "1", userUuid)

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

func TestScoreNotifier_ForbiddenForNonMember(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(0, sql.ErrNoRows)

	h, _ := setupScoreWebsocketTestHandler(t, mockStore)
	server := startScoreWebsocketServer(t, h, "1", userUuid)

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

func TestScoreNotifier_RegisterAndReceiveScoreNotification(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{
		Id:     1,
		Status: model.TEAMS_PLAYING,
	}, nil)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(1, nil)
	mockStore.On("GetDraftScore", mock.Anything, 1).Return([]model.DraftPlayer{}, nil)

	h, notifier := setupScoreWebsocketTestHandler(t, mockStore)
	server := startScoreWebsocketServer(t, h, "1", userUuid)

	conn := dialScoreWebsocket(t, server)

	// Give the handler time to register the watcher
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, notifier.GetWatcherCount(1))

	// Trigger a score notification
	notifier.ReceiveScoreEvent(t.Context(), 1)

	err := conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	require.NoError(t, err)
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.NotEmpty(t, message)

	// Clean disconnect
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, notifier.GetWatcherCount(1))
}

func TestScoreNotifier_IgnorePickNotifications(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{
		Id:     1,
		Status: model.TEAMS_PLAYING,
	}, nil)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(1, nil)

	h, notifier := setupScoreWebsocketTestHandler(t, mockStore)
	server := startScoreWebsocketServer(t, h, "1", userUuid)

	conn := dialScoreWebsocket(t, server)

	time.Sleep(50 * time.Millisecond)

	// Send a pick notification; the score handler should ignore it
	notifier.NotifyWatchers(t.Context(), 1)

	time.Sleep(50 * time.Millisecond)
	mockStore.AssertNotCalled(t, "GetDraftScore", mock.Anything, 1)

	// Clean disconnect
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func TestScoreNotifier_ClosesWhenDraftLeavesTeamsPlaying(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{
		Id:     1,
		Status: model.TEAMS_PLAYING,
	}, nil)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(1, nil)

	h, notifier := setupScoreWebsocketTestHandler(t, mockStore)
	server := startScoreWebsocketServer(t, h, "1", userUuid)

	conn := dialScoreWebsocket(t, server)

	time.Sleep(50 * time.Millisecond)

	// Simulate the draft transitioning to COMPLETE.
	notifier.NotifyStateChange(t.Context(), 1, model.COMPLETE)

	err := conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	require.NoError(t, err)
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
}

func TestScoreNotifier_EnforcesReadLimit(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{
		Id:     1,
		Status: model.TEAMS_PLAYING,
	}, nil)
	mockStore.On("GetDraftPlayerId", mock.Anything, 1, userUuid).Return(1, nil)

	h, _ := setupScoreWebsocketTestHandler(t, mockStore)
	server := startScoreWebsocketServer(t, h, "1", userUuid)
	conn := dialScoreWebsocket(t, server)

	largeMessage := make([]byte, WsMaxMessageSize()+1)
	err := conn.WriteMessage(websocket.BinaryMessage, largeMessage)
	require.NoError(t, err)

	err = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	require.NoError(t, err)
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
}

func TestRenderDraftScoreTable_SortsPlayersAndPicks(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("GetDraftScore", t.Context(), 1).Return([]model.DraftPlayer{
		{
			Id:    2,
			User:  model.User{Username: "player2"},
			Score: 50,
			Picks: []model.Pick{
				{Pick: sql.NullString{String: "frc20", Valid: true}, Score: 10},
				{Pick: sql.NullString{String: "frc10", Valid: true}, Score: 30},
			},
		},
		{
			Id:    1,
			User:  model.User{Username: "player1"},
			Score: 100,
			Picks: []model.Pick{
				{Pick: sql.NullString{String: "frc5", Valid: true}, Score: 40},
				{Pick: sql.NullString{String: "frc1", Valid: true}, Score: 60},
			},
		},
	}, nil)

	h := &Handler{
		Stores:   StorageGroup{DraftStore: mockStore},
		Services: ServiceGroup{AvatarStore: &mockAvatarStore{color: cache.DefaultAvatarColor}},
	}

	html, err := h.renderDraftScoreTable(t.Context(), 1)
	require.NoError(t, err)

	// Players should be sorted by descending total score.
	player1Idx := strings.Index(html, "player1")
	player2Idx := strings.Index(html, "player2")
	require.NotEqual(t, -1, player1Idx)
	require.NotEqual(t, -1, player2Idx)
	assert.Less(t, player1Idx, player2Idx)

	// player1's picks should be sorted by descending individual score (frc1=60 before frc5=40).
	team1LinkIdx := strings.Index(html, "/u/draft/1/team/1")
	team5LinkIdx := strings.Index(html, "/u/draft/1/team/5")
	require.NotEqual(t, -1, team1LinkIdx)
	require.NotEqual(t, -1, team5LinkIdx)
	assert.Less(t, team1LinkIdx, team5LinkIdx)
}
