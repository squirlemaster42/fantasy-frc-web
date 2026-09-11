package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	modelmocks "server/model/mocks"
)

func setupTestContext(t *testing.T, method string, target string, body string, cookieValue string) (*echo.Echo, *echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: cookieValue})
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

	assert.ErrorIs(t, err, errLoginRequired)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestRequireUserUuid_WrongTypeRedirects(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")
	c.Set("userUuid", "not-a-uuid")

	h := &Handler{}
	_, err := h.requireUserUuid(c)

	assert.ErrorIs(t, err, errLoginRequired)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestRequireUserUuid_ValidUuid(t *testing.T) {
	_, c, _ := setupTestContext(t, http.MethodGet, "/u/home", "", "")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	h := &Handler{}
	got, err := h.requireUserUuid(c)

	assert.NoError(t, err)
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

	assert.Error(t, err)
	invokeErrorHandler(echo.New(), err, c)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestInvokeErrorHandler_Helper(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	invokeErrorHandler(e, echo.NewHTTPError(http.StatusBadRequest, "bad"), c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
