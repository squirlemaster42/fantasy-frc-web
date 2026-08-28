package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"server/authentication"
	authmocks "server/authentication/mocks"
)

func TestHandleLoginPost(t *testing.T) {
	t.Run("valid login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=secret&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockAuthService := authmocks.NewMockAuthService(t)

		mockAuthService.On("Login", c.Request().Context(), "testuser", "secret").Return(userUuid, "new-session-token", nil)

		h := &Handler{
			Services: ServiceGroup{
				AuthService: mockAuthService,
			},
			Config: ConfigGroup{
				SecureHttpCookie: true,
			},
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
		assert.Equal(t, "new-session-token", sessionCookie.Value)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=wrong&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockAuthService := authmocks.NewMockAuthService(t)
		mockAuthService.On("Login", c.Request().Context(), "testuser", "wrong").Return(uuid.UUID{}, "", authentication.ErrInvalidCredentials)

		h := &Handler{
			Services: ServiceGroup{
				AuthService: mockAuthService,
			},
			Config: ConfigGroup{
				SecureHttpCookie: true,
			},
		}

		err := h.HandleLoginPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid username or password")
	})

	t.Run("database error validating login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=secret&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockAuthService := authmocks.NewMockAuthService(t)
		mockAuthService.On("Login", c.Request().Context(), "testuser", "secret").Return(uuid.UUID{}, "", errors.New("connection refused"))

		h := &Handler{
			Services: ServiceGroup{
				AuthService: mockAuthService,
			},
			Config: ConfigGroup{
				SecureHttpCookie: true,
			},
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

		mockAuthService := authmocks.NewMockAuthService(t)
		mockAuthService.On("Logout", c.Request().Context(), "test-session").Return(nil)

		h := &Handler{
			Services: ServiceGroup{
				AuthService: mockAuthService,
			},
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
		mockAuthService := authmocks.NewMockAuthService(t)

		mockAuthService.On("Register", c.Request().Context(), "newuser", "Secret123").Return(userUuid, "new-session-token", nil)

		h := &Handler{
			Services: ServiceGroup{
				AuthService: mockAuthService,
			},
			Config: ConfigGroup{
				SecureHttpCookie: true,
			},
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
		assert.Equal(t, "new-session-token", sessionCookie.Value)
	})

	t.Run("username taken", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=existing&password=Secret123&confirmPassword=Secret123&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockAuthService := authmocks.NewMockAuthService(t)
		mockAuthService.On("Register", c.Request().Context(), "existing", "Secret123").Return(uuid.UUID{}, "", authentication.ErrUsernameTaken)

		h := &Handler{
			Services: ServiceGroup{
				AuthService: mockAuthService,
			},
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Username Taken")
	})

	t.Run("passwords do not match", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=newuser&password=Secret123&confirmPassword=Other456&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockAuthService := authmocks.NewMockAuthService(t)
		mockAuthService.On("Register", c.Request().Context(), "newuser", "Secret123").Return(uuid.UUID{}, "", &authentication.ValidationError{Message: "Passwords Do Not Match"})

		h := &Handler{
			Services: ServiceGroup{
				AuthService: mockAuthService,
			},
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Passwords Do Not Match")
	})

	t.Run("database error checking username", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=newuser&password=Secret123&confirmPassword=Secret123&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockAuthService := authmocks.NewMockAuthService(t)
		mockAuthService.On("Register", c.Request().Context(), "newuser", "Secret123").Return(uuid.UUID{}, "", errors.New("connection refused"))

		h := &Handler{
			Services: ServiceGroup{
				AuthService: mockAuthService,
			},
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to create account")
	})

	t.Run("invalid username", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=bad%20user&password=Secret123&confirmPassword=Secret123&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockAuthService := authmocks.NewMockAuthService(t)
		mockAuthService.On("Register", c.Request().Context(), "bad user", "Secret123").Return(uuid.UUID{}, "", &authentication.ValidationError{Message: "Username cannot contain spaces"})

		h := &Handler{
			Services: ServiceGroup{
				AuthService: mockAuthService,
			},
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Username cannot contain spaces")
	})
}

func TestHandleViewLogin(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")

	h := &Handler{}

	err := h.HandleViewLogin(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "<!doctype html>")
	assert.Contains(t, body, `/css/styles.css`)
	assert.Contains(t, body, "Sign In")
}

func TestHandleViewRegister(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/register", "", "")

	h := &Handler{}

	err := h.HandleViewRegister(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "<!doctype html>")
	assert.Contains(t, body, `/css/styles.css`)
	assert.Contains(t, body, "Create Account")
}
