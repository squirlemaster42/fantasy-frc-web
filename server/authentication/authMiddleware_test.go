package authentication

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"server/model/mocks"
)

func TestNewAuth(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	assert.NotNil(t, auth)
	assert.Equal(t, mockUserStore, auth.userStore)
}

func TestAuthenticate_NoSessionCookie(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/home", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := auth.Authenticate(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestAuthenticate_InvalidSession(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/home", nil)
	req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "invalid-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUserStore.On("ValidateSessionToken", c.Request().Context(), "invalid-token").Return(false, nil)

	handler := auth.Authenticate(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
	mockUserStore.AssertExpectations(t)
}

func TestAuthenticate_ValidSession(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/home", nil)
	req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "valid-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUserStore.On("ValidateSessionToken", c.Request().Context(), "valid-token").Return(true, nil)
	mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "valid-token").Return(userUuid, nil)

	var contextUuid uuid.UUID
	handler := auth.Authenticate(func(c echo.Context) error {
		contextUuid = c.Get(string(UserUuidKey)).(uuid.UUID)
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userUuid, contextUuid)
	mockUserStore.AssertExpectations(t)
}

func TestAuthenticate_ValidateSessionError(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/home", nil)
	req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUserStore.On("ValidateSessionToken", c.Request().Context(), "token").Return(false, errors.New("db error"))

	handler := auth.Authenticate(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
	mockUserStore.AssertExpectations(t)
}

func TestAuthenticate_GetUserBySessionError(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/home", nil)
	req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUserStore.On("ValidateSessionToken", c.Request().Context(), "token").Return(true, nil)
	mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "token").Return(uuid.UUID{}, errors.New("db error"))

	handler := auth.Authenticate(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
	mockUserStore.AssertExpectations(t)
}

func TestCheckAdmin_NoUserUuid(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/admin/console", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := auth.CheckAdmin(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/u/home", rec.Header().Get("Location"))
}

func TestCheckAdmin_UserIsAdmin(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/admin/console", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(string(UserUuidKey), userUuid)

	mockUserStore.On("UserIsAdmin", c.Request().Context(), userUuid).Return(true, nil)

	var isAdmin bool
	handler := auth.CheckAdmin(func(c echo.Context) error {
		isAdmin = c.Get(string(IsAdminKey)).(bool)
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, isAdmin)
	mockUserStore.AssertExpectations(t)
}

func TestCheckAdmin_UserIsNotAdmin(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/admin/console", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(string(UserUuidKey), userUuid)

	mockUserStore.On("UserIsAdmin", c.Request().Context(), userUuid).Return(false, nil)

	handler := auth.CheckAdmin(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/u/home", rec.Header().Get("Location"))
	mockUserStore.AssertExpectations(t)
}

func TestCheckAdmin_UserIsAdminError(t *testing.T) {
	mockUserStore := mocks.NewMockUserStore(t)
	auth := NewAuth(mockUserStore)

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/admin/console", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(string(UserUuidKey), userUuid)

	mockUserStore.On("UserIsAdmin", c.Request().Context(), userUuid).Return(false, errors.New("db error"))

	handler := auth.CheckAdmin(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/u/home", rec.Header().Get("Location"))
	mockUserStore.AssertExpectations(t)
}

func TestNewMetricAuth_PanicsOnEmptySecret(t *testing.T) {
	assert.Panics(t, func() {
		NewMetricAuth("")
	})
}

func TestMetricAuth_MetricsAuthMiddleware_MissingHeader(t *testing.T) {
	metricAuth := NewMetricAuth("secret")
	middleware := metricAuth.MetricsAuthMiddleware()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMetricAuth_MetricsAuthMiddleware_MalformedHeader(t *testing.T) {
	metricAuth := NewMetricAuth("secret")
	middleware := metricAuth.MetricsAuthMiddleware()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Basic secret")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMetricAuth_MetricsAuthMiddleware_InvalidToken(t *testing.T) {
	metricAuth := NewMetricAuth("secret")
	middleware := metricAuth.MetricsAuthMiddleware()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMetricAuth_MetricsAuthMiddleware_ValidToken(t *testing.T) {
	metricAuth := NewMetricAuth("secret")
	middleware := metricAuth.MetricsAuthMiddleware()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
