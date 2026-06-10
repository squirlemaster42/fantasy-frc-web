package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"server/model/mocks"
)

func TestHandleLoginPost(t *testing.T) {
	t.Run("valid login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=secret&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "secret").Return(true, nil)
		mockUserStore.On("GetUserUuidByUsername", c.Request().Context(), "testuser").Return(userUuid, nil)
		mockUserStore.On("RegisterSession", c.Request().Context(), userUuid, mock.AnythingOfType("string")).Return(nil)

		h := &Handler{
			UserStore:        mockUserStore,
			SecureHttpCookie: true,
		}

		err := h.HandleLoginPost(c)
		assert.NoError(t, err)
		assert.Equal(t, "/u/home", rec.Header().Get("HX-Redirect"))

		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "sessionToken" {
				sessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, sessionCookie, "sessionToken cookie should be set")
		assert.NotEmpty(t, sessionCookie.Value)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=wrong&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "wrong").Return(false, nil)

		h := &Handler{
			UserStore:        mockUserStore,
			SecureHttpCookie: true,
		}

		err := h.HandleLoginPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "You have entered an invalid username or password")
	})

	t.Run("username not taken", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=newuser&password=secret&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("ValidateLogin", c.Request().Context(), "newuser", "secret").Return(false, nil)

		h := &Handler{
			UserStore:        mockUserStore,
			SecureHttpCookie: true,
		}

		err := h.HandleLoginPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "You have entered an invalid username or password")
	})

	t.Run("database error validating login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=secret&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "secret").Return(false, errors.New("connection refused"))

		h := &Handler{
			UserStore:        mockUserStore,
			SecureHttpCookie: true,
		}

		err := h.HandleLoginPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to validate login")
	})
}

func TestHandleLogoutPost(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/logout", "", "test-session")

		mockUserStore := mocks.NewMockUserStore(t)
		mockUserStore.On("UnRegisterSession", c.Request().Context(), "test-session").Return(nil)

		h := &Handler{
			UserStore: mockUserStore,
		}

		err := h.HandleLogoutPost(c)
		assert.NoError(t, err)
		assert.Equal(t, "/login", rec.Header().Get("HX-Redirect"))

		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "sessionToken" {
				sessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, sessionCookie, "sessionToken cookie should be cleared")
		assert.Empty(t, sessionCookie.Value)
	})
}

func TestHandlerRegisterPost(t *testing.T) {
	t.Run("valid registration", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=newuser&password=Secret123&confirmPassword=Secret123&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, nil)
		mockUserStore.On("RegisterUser", c.Request().Context(), "newuser", "Secret123").Return(userUuid, nil)
		mockUserStore.On("RegisterSession", c.Request().Context(), userUuid, mock.AnythingOfType("string")).Return(nil)

		h := &Handler{
			UserStore:         mockUserStore,
			MinPasswordLength: 8,
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, "/u/home", rec.Header().Get("HX-Redirect"))

		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "sessionToken" {
				sessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, sessionCookie, "sessionToken cookie should be set")
		assert.NotEmpty(t, sessionCookie.Value)
	})

	t.Run("username taken", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=existing&password=Secret123&confirmPassword=Secret123&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "existing").Return(true, nil)

		h := &Handler{
			UserStore:         mockUserStore,
			MinPasswordLength: 8,
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Username Taken")
	})

	t.Run("passwords do not match", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=newuser&password=Secret123&confirmPassword=Other456&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, nil)

		h := &Handler{
			UserStore:         mockUserStore,
			MinPasswordLength: 8,
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Passwords Do Not Match")
	})

	t.Run("database error checking username", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=newuser&password=Secret123&confirmPassword=Secret123&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, errors.New("connection refused"))

		h := &Handler{
			UserStore:         mockUserStore,
			MinPasswordLength: 8,
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to check username availability")
	})
}

func TestHandleViewLogin(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")

	h := &Handler{}

	err := h.HandleViewLogin(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleViewRegister(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/register", "", "")

	h := &Handler{}

	err := h.HandleViewRegister(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// Layer 2 HTML body assertions for auth pages

func TestHandleViewLogin_HTML(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")

	h := &Handler{
		MinPasswordLength: 8,
	}

	err := h.HandleViewLogin(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()

	// HTMX form attributes
	assert.Contains(t, body, `hx-post="/login"`)
	assert.Contains(t, body, `hx-target="#login-box"`)
	assert.Contains(t, body, `hx-swap="outerHTML transition:true"`)

	// Form inputs
	assert.Contains(t, body, `name="username"`)
	assert.Contains(t, body, `name="password"`)
	assert.Contains(t, body, `type="password"`)
	assert.Contains(t, body, `minlength="8"`)

	// CSRF token input
	assert.Contains(t, body, `name="csrf_token"`)
	assert.Contains(t, body, `type="hidden"`)

	// Submit button
	assert.Contains(t, body, `type="submit"`)
	assert.Contains(t, body, `Sign In`)

	// Register link
	assert.Contains(t, body, `href="/register"`)
	assert.Contains(t, body, `Create one here`)

	// Alpine.js loading state
	assert.Contains(t, body, `x-data="{ loading: false }"`)
	assert.Contains(t, body, `@htmx:before-request`)
	assert.Contains(t, body, `x-show="!loading"`)
	assert.Contains(t, body, `x-show="loading"`)

	// Page layout
	assert.Contains(t, body, `<!doctype html>`)
	assert.Contains(t, body, `hx-boost="true"`)
}

func TestHandleViewRegister_HTML(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/register", "", "")

	h := &Handler{
		MinPasswordLength: 8,
	}

	err := h.HandleViewRegister(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()

	// HTMX form attributes
	assert.Contains(t, body, `hx-post="/register"`)
	assert.Contains(t, body, `hx-target="#register-box"`)

	// Form inputs
	assert.Contains(t, body, `name="username"`)
	assert.Contains(t, body, `name="password"`)
	assert.Contains(t, body, `name="confirmPassword"`)
	assert.Contains(t, body, `type="password"`)
	assert.Contains(t, body, `minlength="8"`)

	// CSRF token input
	assert.Contains(t, body, `name="csrf_token"`)
	assert.Contains(t, body, `type="hidden"`)

	// Submit button
	assert.Contains(t, body, `type="submit"`)
	assert.Contains(t, body, `Create Account`)

	// Login link
	assert.Contains(t, body, `href="/login"`)
	assert.Contains(t, body, `Sign in here`)

	// Page layout
	assert.Contains(t, body, `<!doctype html>`)
	assert.Contains(t, body, `hx-boost="true"`)
}

func TestHandleLoginPost_InvalidCredentials_HTML(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=wrong&csrf_token=test-csrf", "")
	c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

	mockUserStore := mocks.NewMockUserStore(t)
	mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "wrong").Return(false, nil)

	h := &Handler{
		UserStore:        mockUserStore,
		SecureHttpCookie: true,
		MinPasswordLength: 8,
	}

	err := h.HandleLoginPost(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()

	// Error message should appear in the response body
	assert.Contains(t, body, "You have entered an invalid username or password")
	// The response should be the login form (since it's an HTMX swap)
	assert.Contains(t, body, `hx-post="/login"`)
	assert.Contains(t, body, `name="username"`)
}

func TestHandleLoginPost_ValidLogin_Headers(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=secret&csrf_token=test-csrf", "")
	c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockUserStore := mocks.NewMockUserStore(t)

	mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "secret").Return(true, nil)
	mockUserStore.On("GetUserUuidByUsername", c.Request().Context(), "testuser").Return(userUuid, nil)
	mockUserStore.On("RegisterSession", c.Request().Context(), userUuid, mock.AnythingOfType("string")).Return(nil)

	h := &Handler{
		UserStore:        mockUserStore,
		SecureHttpCookie: true,
	}

	err := h.HandleLoginPost(c)
	assert.NoError(t, err)

	// HTMX redirect header
	assert.Equal(t, "/u/home", rec.Header().Get("HX-Redirect"))

	// Session cookie set
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "sessionToken" {
			sessionCookie = cookie
			break
		}
	}
	assert.NotNil(t, sessionCookie, "sessionToken cookie should be set")
	assert.NotEmpty(t, sessionCookie.Value)
	assert.Equal(t, "/", sessionCookie.Path)
}
