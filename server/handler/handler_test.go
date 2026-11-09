package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"server/model"

	"uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	modelmocks "server/model/mocks"
)

func setupTestContext(t *testing.T, method string, target string, body string, cookieValue string) (*echo.Echo, *echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequestWithContext(context.Background(), method, target, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	} else {
		req = httptest.NewRequestWithContext(context.Background(), method, target, nil)
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: cookieValue}) //nolint:gosec // test cookie
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return e, c, rec
}

// invokeErrorHandler runs the Echo error handler for an error. This helper
// centralizes the Echo v5 signature swap so call sites stay stable.
func invokeErrorHandler(e *echo.Echo, err error, c *echo.Context) {
	e.HTTPErrorHandler(c, err)
}

func TestRequireUserUuid_MissingUuidRedirects(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	h := &Handler{}
	_, err := h.requireUserUuid(c)

	require.ErrorIs(t, err, errLoginRequired)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestRequireUserUuid_WrongTypeRedirects(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")
	c.Set("userUuid", "not-a-uuid")

	h := &Handler{}
	_, err := h.requireUserUuid(c)

	require.ErrorIs(t, err, errLoginRequired)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestRequireUserUuid_ValidUuid(t *testing.T) {
	_, c, _ := setupTestContext(t, http.MethodGet, "/u/home", "", "")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	h := &Handler{}
	got, err := h.requireUserUuid(c)

	require.NoError(t, err)
	assert.Equal(t, userUuid, got)
}

func TestGetAuthenticatedUsername_StoreErrorReturns500(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockUserStore := modelmocks.NewMockUserStore(t)
	mockUserStore.On("GetUsername", mock.Anything, userUuid).Return("", errors.New("db down"))

	h := &Handler{
		Stores: StorageGroup{
			UserStore: mockUserStore,
		},
	}

	_, err := h.getAuthenticatedUsername(c, userUuid)

	require.Error(t, err)
	invokeErrorHandler(echo.New(), err, c)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestInvokeErrorHandler_Helper(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	invokeErrorHandler(e, echo.NewHTTPError(http.StatusBadRequest, "bad"), c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestParseDraftId_Valid(t *testing.T) {
	_, c, _ := setupTestContext(t, http.MethodGet, "/u/draft/42/profile", "", "")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

	h := &Handler{}
	draftId, err := h.parseDraftId(c)

	require.NoError(t, err)
	assert.Equal(t, 42, draftId)
}

func TestParseDraftId_Invalid(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/abc/profile", "", "")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})

	h := &Handler{}
	draftId, err := h.parseDraftId(c)

	require.Error(t, err)
	assert.Equal(t, 0, draftId)
	// parseDraftId itself does not write the response; it returns an error.
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestParseDraftId_Missing(t *testing.T) {
	_, c, _ := setupTestContext(t, http.MethodGet, "/u/draft//profile", "", "")
	// No path value set.

	h := &Handler{}
	draftId, err := h.parseDraftId(c)

	require.Error(t, err)
	assert.Equal(t, 0, draftId)
}

func TestRequireDraftOwnerAdmin_Success(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/admin/endDraft", "", "")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockDraftStore := modelmocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{
		Id:    42,
		Owner: model.User{UserUuid: userUuid},
	}, nil)

	h := &Handler{Stores: StorageGroup{DraftStore: mockDraftStore}}
	gotUuid, draftModel, ok := h.requireDraftOwnerAdmin(c)

	require.True(t, ok)
	assert.Equal(t, userUuid, gotUuid)
	assert.Equal(t, 42, draftModel.Id)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireDraftOwnerAdmin_InvalidDraftId(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/abc/admin/endDraft", "", "")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "abc"}})
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	h := &Handler{}
	_, _, ok := h.requireDraftOwnerAdmin(c)

	require.False(t, ok)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Response body contains the rendered admin message.
	assert.Contains(t, rec.Body.String(), "Invalid draft ID")
}

func TestRequireDraftOwnerAdmin_DraftNotFound(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/admin/endDraft", "", "")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockDraftStore := modelmocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{}, errors.New("db down"))

	h := &Handler{Stores: StorageGroup{DraftStore: mockDraftStore}}
	_, _, ok := h.requireDraftOwnerAdmin(c)

	require.False(t, ok)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Draft not found")
}

func TestRequireDraftOwnerAdmin_NonOwner(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/admin/endDraft", "", "")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	ownerUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	c.Set("userUuid", userUuid)

	mockDraftStore := modelmocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", mock.Anything, 42).Return(model.DraftModel{
		Id:    42,
		Owner: model.User{UserUuid: ownerUuid},
	}, nil)

	h := &Handler{Stores: StorageGroup{DraftStore: mockDraftStore}}
	_, _, ok := h.requireDraftOwnerAdmin(c)

	require.False(t, ok)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Permission denied")
}

func TestRequireDraftOwnerAdmin_MissingUserUuid(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/u/draft/42/admin/endDraft", "", "")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "42"}})

	h := &Handler{}
	_, _, ok := h.requireDraftOwnerAdmin(c)

	require.False(t, ok)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}
