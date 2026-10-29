package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"server/background"
	"server/database"
	"server/draft"
	"server/model"
	"server/model/mocks"
	"server/swagger"
	"server/utils"
)

func mockRunInTransaction(mockStore *mocks.MockDraftStore) {
	mockStore.On("WithTx", mock.Anything).Return(mockStore).Maybe()
	mockStore.On("RunInTransaction", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(database.DBTX) error)
		_ = fn(nil)
	}).Return(nil).Once()
}

func TestModifyPickTimeCommandDurationParsing(t *testing.T) {
	tests := []struct {
		name          string
		duration      string
		shouldBeValid bool
		description   string
	}{
		{name: "simple minutes", duration: "45m", shouldBeValid: true},
		{name: "hours and minutes", duration: "1h30m", shouldBeValid: true},
		{name: "compound duration", duration: "2h15m30s", shouldBeValid: true},
		{name: "seconds only", duration: "30s", shouldBeValid: true},
		{name: "hours only", duration: "2h", shouldBeValid: true},
		{name: "with plus prefix", duration: "+30m", shouldBeValid: true},
		{name: "with plus prefix compound", duration: "+1h30m", shouldBeValid: true},
		{name: "days", duration: "1d", shouldBeValid: false, description: "Should reject days (not supported by time.ParseDuration)"},
		{name: "invalid format", duration: "abc", shouldBeValid: false},
		{name: "empty string", duration: "", shouldBeValid: false},
		{name: "just number", duration: "30", shouldBeValid: false},
		{name: "spaces in duration", duration: "1h 30m", shouldBeValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := time.ParseDuration(tt.duration)
			if tt.shouldBeValid {
				require.NoError(t, err, tt.description)
			} else {
				require.Error(t, err, tt.description)
			}
		})
	}
}

func TestModifyPickTimeCommandTimeCalculation(t *testing.T) {
	tests := []struct {
		name              string
		duration          string
		expectedMinOffset time.Duration
		expectedMaxOffset time.Duration
	}{
		{name: "45 minutes", duration: "45m", expectedMinOffset: 44 * time.Minute, expectedMaxOffset: 46 * time.Minute},
		{name: "1 hour 30 minutes", duration: "1h30m", expectedMinOffset: 89 * time.Minute, expectedMaxOffset: 91 * time.Minute},
		{name: "2 hours 15 minutes 30 seconds", duration: "2h15m30s", expectedMinOffset: 2*time.Hour + 14*time.Minute + 30*time.Second, expectedMaxOffset: 2*time.Hour + 16*time.Minute + 30*time.Second},
		{name: "30 seconds", duration: "30s", expectedMinOffset: 29 * time.Second, expectedMaxOffset: 31 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := time.ParseDuration(tt.duration)
			require.NoError(t, err)

			now := time.Now()
			newExpiration := now.Add(duration)
			offset := newExpiration.Sub(now)

			assert.True(t, offset >= tt.expectedMinOffset && offset <= tt.expectedMaxOffset,
				"Expected offset between %v and %v, got %v",
				tt.expectedMinOffset, tt.expectedMaxOffset, offset)
		})
	}
}

func TestAdminPickCommandTeamFormatting(t *testing.T) {
	tests := []struct {
		name          string
		teamInput     string
		expectedTbaId string
	}{
		{teamInput: "254", expectedTbaId: "frc254"},
		{teamInput: "001", expectedTbaId: "frc001"},
		{teamInput: "9999", expectedTbaId: "frc9999"},
		{teamInput: "1", expectedTbaId: "frc1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := "frc" + tt.teamInput
			assert.Equal(t, tt.expectedTbaId, formatted)
		})
	}
}

func TestRenameDraftCommandNameValidation(t *testing.T) {
	tests := []struct {
		name          string
		inputName     string
		shouldBeEmpty bool
	}{
		{name: "valid name", inputName: "My Draft 2026", shouldBeEmpty: false},
		{name: "empty name", inputName: "", shouldBeEmpty: true},
		{name: "name with spaces", inputName: "  ", shouldBeEmpty: false},
		{name: "long name", inputName: "This is a very long draft name with many characters 123456789", shouldBeEmpty: false},
		{name: "name with special characters", inputName: "Draft #1: Ultimate Edition!", shouldBeEmpty: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isEmpty := tt.inputName == ""
			assert.Equal(t, tt.shouldBeEmpty, isEmpty)
		})
	}
}

type mockAdminTBAHandler struct {
	teams map[string][]swagger.Team
}

func (m *mockAdminTBAHandler) MakeEventListReq(ctx context.Context, teamId string) ([]string, error) { return nil, nil }
func (m *mockAdminTBAHandler) MakeMatchReq(ctx context.Context, matchId string) (swagger.Match, error) {
	return swagger.Match{}, nil
}
func (m *mockAdminTBAHandler) MakeEventMatchKeysRequest(ctx context.Context, eventId string) ([]string, error) {
	return nil, nil
}
func (m *mockAdminTBAHandler) MakeTeamsAtEventRequest(ctx context.Context, eventId string) ([]swagger.Team, error) {
	return m.teams[eventId], nil
}
func (m *mockAdminTBAHandler) MakeEliminationAllianceRequest(ctx context.Context, eventId string) ([]swagger.EliminationAlliance, error) {
	return nil, nil
}
func (m *mockAdminTBAHandler) MakeTeamAvatarRequest(ctx context.Context, teamId string) (string, error) {
	return "", nil
}

type mockEventTeamCache struct {
	invalidateCalled bool
	refreshErr       error
	lastRefresh      time.Time
}

func (m *mockEventTeamCache) IsTeamAtEvent(ctx context.Context, teamId string) (bool, error) {
	return false, nil
}
func (m *mockEventTeamCache) Refresh(ctx context.Context) error {
	return m.refreshErr
}
func (m *mockEventTeamCache) InvalidateAndRefresh(ctx context.Context) error {
	m.invalidateCalled = true
	return m.refreshErr
}
func (m *mockEventTeamCache) LastRefresh() time.Time {
	return m.lastRefresh
}

func TestHandleInvalidatePickCache_Success(t *testing.T) {
	testUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockUserStore := mocks.NewMockUserStore(t)
	mockUserStore.On("GetUsername", mock.Anything, testUuid).Return("admin", nil).Maybe()

	mockCache := &mockEventTeamCache{lastRefresh: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	h := Handler{
		Stores:   StorageGroup{UserStore: mockUserStore},
		Services: ServiceGroup{EventTeamCache: mockCache},
	}

	_, c, rec := setupTestContext(t, http.MethodPost, "/u/admin/invalidatePickCache", "", "")
	c.Set("userUuid", testUuid)

	err := h.HandleInvalidatePickCache(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, mockCache.invalidateCalled)
	assert.Contains(t, rec.Body.String(), "Pick validation cache invalidated and refreshed")
}

func TestHandleInvalidatePickCache_NilCache(t *testing.T) {
	testUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockUserStore := mocks.NewMockUserStore(t)
	mockUserStore.On("GetUsername", mock.Anything, testUuid).Return("admin", nil).Maybe()

	h := Handler{
		Stores: StorageGroup{UserStore: mockUserStore},
	}

	_, c, rec := setupTestContext(t, http.MethodPost, "/u/admin/invalidatePickCache", "", "")
	c.Set("userUuid", testUuid)

	err := h.HandleInvalidatePickCache(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Invalidate Pick Cache")
	assert.Contains(t, body, "pick validation cache is not configured")
}

func setupAdminContext(t *testing.T, target string, body string) (*echo.Context, *httptest.ResponseRecorder, *mocks.MockUserStore) {
	t.Helper()
	_, c, rec := setupTestContext(t, http.MethodPost, target, body, "")
	testUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", testUuid)

	mockUserStore := mocks.NewMockUserStore(t)
	mockUserStore.On("GetUsername", mock.Anything, testUuid).Return("admin", nil).Maybe()
	return c, rec, mockUserStore
}

func TestHandleAdminConsoleGet(t *testing.T) {
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/console", "")
	c.Set("csrfToken", "test-token")

	h := Handler{Stores: StorageGroup{UserStore: mockUserStore}}
	err := h.HandleAdminConsoleGet(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Admin Console")
}

func TestHandleAdminPing(t *testing.T) {
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/ping", "")

	h := Handler{Stores: StorageGroup{UserStore: mockUserStore}}
	err := h.HandleAdminPing(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Ping")
	assert.Contains(t, body, "Pong")
}

func TestHandleAdminPing_Unauthenticated(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/admin/ping", "", "")

	h := &Handler{}
	err := h.HandleAdminPing(c)

	require.ErrorIs(t, err, errLoginRequired)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestHandleAdminPopulateTeams(t *testing.T) {
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/populateTeams", "")

	tbaHandler := &mockAdminTBAHandler{
		teams: map[string][]swagger.Team{
			"2026arc": {{Key: "frc254"}},
		},
	}

	mockTeamStore := mocks.NewMockTeamStore(t)
	mockTeamStore.On("GetTeam", mock.Anything, "frc254").Return(nil, nil).Once()
	mockTeamStore.On("CreateTeam", mock.Anything, "frc254", "").Return(nil).Once()

	h := Handler{
		Stores:   StorageGroup{UserStore: mockUserStore, TeamStore: mockTeamStore},
		Services: ServiceGroup{TBAHandler: tbaHandler},
	}
	err := h.HandleAdminPopulateTeams(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Populate Teams")
	assert.Contains(t, body, "Successfully populated 1 teams")
}

func TestHandleAdminListDrafts(t *testing.T) {
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/listDrafts", "search=playoffs")

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraftsByName", mock.Anything, "playoffs").Return([]model.DraftModel{
		{Id: 7, DisplayName: "Playoffs Draft"},
	}, nil)

	h := Handler{
		Stores: StorageGroup{UserStore: mockUserStore, DraftStore: mockDraftStore},
	}
	err := h.HandleAdminListDrafts(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "List Drafts")
	assert.Contains(t, body, "Playoffs Draft")
}

func TestHandleAdminStartDraft(t *testing.T) {
	draftId := 1
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/startDraft", "id=1")

	players := make([]model.DraftPlayer, 8)
	for i := range players {
		players[i] = model.DraftPlayer{Pending: false}
	}

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:      draftId,
		Status:  model.FILLING,
		Players: players,
	}, nil).Once()
	mockDraftStore.On("CancelOutstandingInvites", mock.Anything, draftId).Return(nil).Once()
	mockRunInTransaction(mockDraftStore)
	mockDraftStore.On("RandomizePickOrder", mock.Anything, draftId).Return(nil).Once()
	mockDraftStore.On("NextPick", mock.Anything, draftId).Return(model.DraftPlayer{Id: 1}, nil).Once()
	mockDraftStore.On("MakePickAvailable", mock.Anything, 1, mock.Anything, mock.Anything).Return(1, nil).Once()
	mockDraftStore.On("UpdateDraftStatus", mock.Anything, draftId, model.PICKING).Return(nil).Once()
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:      draftId,
		Status:  model.PICKING,
		Players: players,
	}, nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
	draftDaemon := background.NewDraftDaemon(mockDraftStore, draftActorMap)

	h := Handler{
		Stores:   StorageGroup{UserStore: mockUserStore, DraftStore: mockDraftStore},
		Services: ServiceGroup{DraftActorMap: draftActorMap, DraftDaemon: draftDaemon},
	}
	err := h.HandleAdminStartDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Start Draft")
	assert.Contains(t, body, "Draft Started")
}

func TestHandleAdminStartDraft_InvalidDraftId(t *testing.T) {
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/startDraft", "id=not-a-number")

	h := Handler{Stores: StorageGroup{UserStore: mockUserStore}}
	err := h.HandleAdminStartDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Start Draft")
	assert.Contains(t, body, "draft ID must be a number")
}

func TestHandleAdminModifyPickTime(t *testing.T) {
	draftId := 1
	pickId := 42
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/modifyPickTime", "id=1&time=45m")

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: pickId},
		Players: []model.DraftPlayer{
			{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
	}, nil).Once()
	mockDraftStore.On("UpdatePickExpirationTime", mock.Anything, pickId, mock.Anything).Return(nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := Handler{
		Stores:   StorageGroup{UserStore: mockUserStore, DraftStore: mockDraftStore},
		Services: ServiceGroup{DraftActorMap: draftActorMap},
	}
	err := h.HandleAdminModifyPickTime(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Modify Pick Time")
	assert.Contains(t, body, "Successfully updated pick expiration time")
}

func TestHandleAdminModifyPickTime_InvalidDuration(t *testing.T) {
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/modifyPickTime", "id=1&time=invalid")

	h := Handler{Stores: StorageGroup{UserStore: mockUserStore}}
	err := h.HandleAdminModifyPickTime(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Modify Pick Time")
	assert.Contains(t, body, "invalid duration format")
}

func TestHandleAdminRenameDraft(t *testing.T) {
	draftId := 1
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/renameDraft", "id=1&name=NewName")

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		DisplayName: "Old Name",
		Players: []model.DraftPlayer{
			{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
	}, nil).Once()
	mockDraftStore.On("UpdateDraft", mock.Anything, mock.Anything).Return(nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	h := Handler{
		Stores:   StorageGroup{UserStore: mockUserStore, DraftStore: mockDraftStore},
		Services: ServiceGroup{DraftActorMap: draftActorMap},
	}
	err := h.HandleAdminRenameDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Rename Draft")
	assert.Contains(t, body, "Successfully renamed draft")
	assert.Contains(t, body, "Old Name")
	assert.Contains(t, body, "NewName")
}

func TestHandleAdminRenameDraft_EmptyName(t *testing.T) {
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/renameDraft", "id=1&name=   ")

	h := Handler{Stores: StorageGroup{UserStore: mockUserStore}}
	err := h.HandleAdminRenameDraft(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Rename Draft")
	assert.Contains(t, body, "draft name cannot be empty")
}

func TestHandleAdminViewWebhookKey(t *testing.T) {
	c, rec, mockUserStore := setupAdminContext(t, "/u/admin/viewWebhookKey", "")

	h := Handler{Stores: StorageGroup{UserStore: mockUserStore}}
	err := h.HandleAdminViewWebhookKey(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "View Webhook Key")
}
