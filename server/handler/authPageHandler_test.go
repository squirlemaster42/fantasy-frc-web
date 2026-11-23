package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"server/authentication"
	authmocks "server/authentication/mocks"
	"server/middleware"
	"server/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"uuid"
)

func TestSetSessionCookie(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")

	h := &Handler{
		Config: ConfigGroup{
			SecureHttpCookie: true,
		},
	}

	sessionToken := "test-session-token"
	h.setSessionCookie(c, sessionToken)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, authentication.SessionCookieName, cookie.Name)
	assert.Equal(t, sessionToken, cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)
	assert.Equal(t, model.SessionExpirationDays()*86400, cookie.MaxAge)
}

func TestSetSessionCookie_InsecureWhenConfigured(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")

	h := &Handler{
		Config: ConfigGroup{
			SecureHttpCookie: false,
		},
	}

	h.setSessionCookie(c, "token")

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.False(t, cookies[0].Secure)
}

func TestHandleViewLogin(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")

	h := &Handler{Config: ConfigGroup{MinPasswordLength: 12}}
	err := h.HandleViewLogin(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Sign In")
	assertCsrfCookie(t, rec)
}

func TestHandleViewRegister(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/register", "", "")

	h := &Handler{Config: ConfigGroup{MinPasswordLength: 12}}
	err := h.HandleViewRegister(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Create Account")
	assertCsrfCookie(t, rec)
}

func TestHandleViewLogin_RenderError(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")
	c.SetResponse(&failingResponseWriter{ResponseWriter: rec})

	h := &Handler{Config: ConfigGroup{MinPasswordLength: 12}}
	err := h.HandleViewLogin(c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}

func TestHandleViewRegister_RenderError(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/register", "", "")
	c.SetResponse(&failingResponseWriter{ResponseWriter: rec})

	h := &Handler{Config: ConfigGroup{MinPasswordLength: 12}}
	err := h.HandleViewRegister(c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}

func TestRenderLoginWithError_RenderError(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")
	c.SetResponse(&failingResponseWriter{ResponseWriter: rec})

	h := &Handler{Config: ConfigGroup{MinPasswordLength: 12}}
	err := h.renderLoginWithError(c, "bad credentials")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}

func TestRenderRegisterWithError_RenderError(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/register", "", "")
	c.SetResponse(&failingResponseWriter{ResponseWriter: rec})

	h := &Handler{Config: ConfigGroup{MinPasswordLength: 12}}
	err := h.renderRegisterWithError(c, "username taken")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}

func TestHandleLoginPost_Success(t *testing.T) {
	csrfValue := "dummy-csrf-value"
	userUuid := uuid.New()
	sessionToken := "test-session-token"

	mockAuthService := authmocks.NewMockAuthService(t)
	mockAuthService.On("Login", mock.Anything, "testuser", "Password123!").
		Return(userUuid, sessionToken, nil)

	body := loginFormBody("testuser", "Password123!", csrfValue)
	_, c, rec := setupTestContext(t, http.MethodPost, "/login", body, "")
	c.Request().AddCookie(testCsrfCookie(csrfValue))

	h := &Handler{
		Config:   ConfigGroup{MinPasswordLength: 12},
		Services: ServiceGroup{AuthService: mockAuthService},
	}
	err := h.HandleLoginPost(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/u/home", rec.Header().Get("HX-Redirect"))
	assertSessionCookie(t, rec, sessionToken)
}

func TestHandleLoginPost_InvalidCredentials(t *testing.T) {
	csrfValue := "dummy-csrf-value"

	mockAuthService := authmocks.NewMockAuthService(t)
	mockAuthService.On("Login", mock.Anything, "testuser", "Password123!").
		Return(uuid.UUID{}, "", authentication.ErrInvalidCredentials)

	body := loginFormBody("testuser", "Password123!", csrfValue)
	_, c, rec := setupTestContext(t, http.MethodPost, "/login", body, "")
	c.Request().AddCookie(testCsrfCookie(csrfValue))

	h := &Handler{
		Config:   ConfigGroup{MinPasswordLength: 12},
		Services: ServiceGroup{AuthService: mockAuthService},
	}
	err := h.HandleLoginPost(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid username or password")
}

func TestHandleLoginPost_CSRFFailure(t *testing.T) {
	body := loginFormBody("testuser", "Password123!", "submitted-value")
	_, c, rec := setupTestContext(t, http.MethodPost, "/login", body, "")
	c.Request().AddCookie(testCsrfCookie("cookie-value"))

	h := &Handler{Config: ConfigGroup{MinPasswordLength: 12}}
	err := h.HandleLoginPost(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid request")
}

func TestHandleLoginPost_AuthServiceError(t *testing.T) {
	csrfValue := "dummy-csrf-value"

	mockAuthService := authmocks.NewMockAuthService(t)
	mockAuthService.On("Login", mock.Anything, "testuser", "Password123!").
		Return(uuid.UUID{}, "", errors.New("db down"))

	body := loginFormBody("testuser", "Password123!", csrfValue)
	_, c, rec := setupTestContext(t, http.MethodPost, "/login", body, "")
	c.Request().AddCookie(testCsrfCookie(csrfValue))

	h := &Handler{
		Config:   ConfigGroup{MinPasswordLength: 12},
		Services: ServiceGroup{AuthService: mockAuthService},
	}
	err := h.HandleLoginPost(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "An error occurred")
}

func TestHandleLogoutPost(t *testing.T) {
	userUuid := uuid.New()
	sessionToken := "test-session-token"

	mockAuthService := authmocks.NewMockAuthService(t)
	mockAuthService.On("Logout", mock.Anything, sessionToken).Return(nil)

	_, c, rec := setupTestContext(t, http.MethodPost, "/logout", "", sessionToken)
	c.Set("userUuid", userUuid)

	h := &Handler{
		Config:   ConfigGroup{SecureHttpCookie: true},
		Services: ServiceGroup{AuthService: mockAuthService},
	}
	err := h.HandleLogoutPost(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("HX-Redirect"))

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, authentication.SessionCookieName, cookies[0].Name)
	assert.Empty(t, cookies[0].Value)
	assert.Equal(t, -1, cookies[0].MaxAge)
}

func TestHandlerRegisterPost_Success(t *testing.T) {
	csrfValue := "dummy-csrf-value"
	userUuid := uuid.New()
	sessionToken := "test-session-token"

	mockAuthService := authmocks.NewMockAuthService(t)
	mockAuthService.On("Register", mock.Anything, "newuser", "Password123!", "Password123!").
		Return(userUuid, sessionToken, nil)

	body := registerFormBody("newuser", "Password123!", "Password123!", csrfValue)
	_, c, rec := setupTestContext(t, http.MethodPost, "/register", body, "")
	c.Request().AddCookie(testCsrfCookie(csrfValue))

	h := &Handler{
		Config:   ConfigGroup{MinPasswordLength: 12},
		Services: ServiceGroup{AuthService: mockAuthService},
	}
	err := h.HandlerRegisterPost(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/u/home", rec.Header().Get("HX-Redirect"))
	assertSessionCookie(t, rec, sessionToken)
}

func TestHandlerRegisterPost_UsernameTaken(t *testing.T) {
	csrfValue := "dummy-csrf-value"

	mockAuthService := authmocks.NewMockAuthService(t)
	mockAuthService.On("Register", mock.Anything, "newuser", "Password123!", "Password123!").
		Return(uuid.UUID{}, "", authentication.ErrUsernameTaken)

	body := registerFormBody("newuser", "Password123!", "Password123!", csrfValue)
	_, c, rec := setupTestContext(t, http.MethodPost, "/register", body, "")
	c.Request().AddCookie(testCsrfCookie(csrfValue))

	h := &Handler{
		Config:   ConfigGroup{MinPasswordLength: 12},
		Services: ServiceGroup{AuthService: mockAuthService},
	}
	err := h.HandlerRegisterPost(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Username is already taken")
}

func TestHandlerRegisterPost_CSRFFailure(t *testing.T) {
	body := registerFormBody("newuser", "Password123!", "Password123!", "submitted-value")
	_, c, rec := setupTestContext(t, http.MethodPost, "/register", body, "")
	c.Request().AddCookie(testCsrfCookie("cookie-value"))

	h := &Handler{Config: ConfigGroup{MinPasswordLength: 12}}
	err := h.HandlerRegisterPost(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid request")
}

func TestHandlerRegisterPost_AuthServiceError(t *testing.T) {
	csrfValue := "dummy-csrf-value"

	mockAuthService := authmocks.NewMockAuthService(t)
	mockAuthService.On("Register", mock.Anything, "newuser", "Password123!", "Password123!").
		Return(uuid.UUID{}, "", errors.New("db down"))

	body := registerFormBody("newuser", "Password123!", "Password123!", csrfValue)
	_, c, rec := setupTestContext(t, http.MethodPost, "/register", body, "")
	c.Request().AddCookie(testCsrfCookie(csrfValue))

	h := &Handler{
		Config:   ConfigGroup{MinPasswordLength: 12},
		Services: ServiceGroup{AuthService: mockAuthService},
	}
	err := h.HandlerRegisterPost(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "An error occurred")
}

func assertCsrfCookie(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == middleware.CsrfCookieName {
			assert.NotEmpty(t, cookie.Value)
			return
		}
	}
	t.Fatalf("expected %s cookie to be set", middleware.CsrfCookieName)
}

func assertSessionCookie(t *testing.T, rec *httptest.ResponseRecorder, value string) {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == authentication.SessionCookieName {
			assert.Equal(t, value, cookie.Value)
			return
		}
	}
	t.Fatalf("expected %s cookie to be set", authentication.SessionCookieName)
}

func testCsrfCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     middleware.CsrfCookieName,
		Value:    value,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
}

func loginFormBody(username, password, csrfValue string) string {
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	form.Set(middleware.CsrfTokenFieldName, csrfValue)
	return form.Encode()
}

func registerFormBody(username, password, confirmPassword, csrfValue string) string {
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	form.Set("confirm_password", confirmPassword)
	form.Set(middleware.CsrfTokenFieldName, csrfValue)
	return form.Encode()
}
