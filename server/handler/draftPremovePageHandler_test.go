package handler

import (
	"net/http"
	"net/url"
	"testing"

	"uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"server/model/mocks"
)

func TestNormalizeTeamNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "plain number", input: "254", want: "frc254"},
		{name: "prefixed", input: "frc254", want: "frc254"},
		{name: "uppercase prefix", input: "FRC254", want: "frc254"},
		{name: "surrounding whitespace", input: "  1678  ", want: "frc1678"},
		{name: "leading zeros", input: "00254", want: "frc254"},
		{name: "empty", input: "", wantErr: true},
		{name: "prefix only", input: "frc", wantErr: true},
		{name: "not a number", input: "254a", wantErr: true},
		{name: "negative", input: "-254", wantErr: true},
		{name: "zero", input: "0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeTeamNumber(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizePremoveTeams(t *testing.T) {
	teams, err := normalizePremoveTeams([]string{"254", "frc1678", "FRC254"})
	require.Error(t, err)
	require.Nil(t, teams)

	teams, err = normalizePremoveTeams([]string{"254", "frc1678"})
	require.NoError(t, err)
	require.Equal(t, []string{"frc254", "frc1678"}, teams)

	teams, err = normalizePremoveTeams(nil)
	require.NoError(t, err)
	require.Empty(t, teams)
}

func TestNormalizePremoveTeamsRejectsTooMany(t *testing.T) {
	values := make([]string, 9)
	for i := range values {
		values[i] = "254"
	}

	_, err := normalizePremoveTeams(values)

	require.Error(t, err)
}

func TestHandleSavePremoves(t *testing.T) {
	body := url.Values{"premoves": {"254", "frc1678"}}.Encode()
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/premoves", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetDraftPlayerId", c.Request().Context(), 42, userUuid).Return(7, nil)
	mockDraftStore.On("ReplacePremoves", c.Request().Context(), 7, []string{"frc254", "frc1678"}).Return(nil, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		},
	}

	err := h.HandleSavePremoves(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/u/draft/42/premoves?saved=1", rec.Header().Get("Location"))
}

func TestHandleSavePremovesClear(t *testing.T) {
	body := url.Values{"action": {"clear"}}.Encode()
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/premoves", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetDraftPlayerId", c.Request().Context(), 42, userUuid).Return(7, nil)
	mockDraftStore.On("ReplacePremoves", c.Request().Context(), 7, []string(nil)).Return(nil, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		},
	}

	err := h.HandleSavePremoves(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/u/draft/42/premoves?saved=1", rec.Header().Get("Location"))
}

func TestHandleSavePremovesBeaconUnavailableTeam(t *testing.T) {
	body := url.Values{"premoves": {"1699"}, "beacon": {"1"}}.Encode()
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/premoves", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetDraftPlayerId", c.Request().Context(), 42, userUuid).Return(7, nil)
	mockDraftStore.On("ReplacePremoves", c.Request().Context(), 7, []string{"frc1699"}).Return([]string{"frc1699"}, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		},
	}

	err := h.HandleSavePremoves(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandleSavePremovesWarnsAboutUnavailableTeams(t *testing.T) {
	body := url.Values{"premoves": {"254", "1699"}}.Encode()
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/premoves", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetDraftPlayerId", c.Request().Context(), 42, userUuid).Return(7, nil)
	mockDraftStore.On("ReplacePremoves", c.Request().Context(), 7, []string{"frc254", "frc1699"}).Return([]string{"frc1699"}, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		},
	}

	err := h.HandleSavePremoves(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/u/draft/42/premoves?saved=1&unavailable=frc1699", rec.Header().Get("Location"))
}

func TestHandleSavePremovesBeacon(t *testing.T) {
	body := url.Values{"premoves": {"254"}, "beacon": {"1"}}.Encode()
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/premoves", body, "test-session")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetDraftPlayerId", c.Request().Context(), 42, userUuid).Return(7, nil)
	mockDraftStore.On("ReplacePremoves", c.Request().Context(), 7, []string{"frc254"}).Return(nil, nil)

	h := &Handler{
		Stores: StorageGroup{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		},
	}

	err := h.HandleSavePremoves(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, rec.Code)
}
