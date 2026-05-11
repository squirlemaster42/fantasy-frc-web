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
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=secret", "")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "testuser").Return(true, nil)
		mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "secret").Return(true)
		mockUserStore.On("GetUserUuidByUsername", c.Request().Context(), "testuser").Return(userUuid)
		mockUserStore.On("RegisterSession", c.Request().Context(), userUuid, mock.AnythingOfType("string"))

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
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=wrong", "")

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "testuser").Return(true, nil)
		mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "wrong").Return(false)

		h := &Handler{
			UserStore:        mockUserStore,
			SecureHttpCookie: true,
		}

		err := h.HandleLoginPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid username or password")
	})

	t.Run("username not taken", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=newuser&password=secret", "")

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, nil)

		h := &Handler{
			UserStore:        mockUserStore,
			SecureHttpCookie: true,
		}

		err := h.HandleLoginPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid username or password")
	})

	t.Run("database error checking username", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=secret", "")

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "testuser").Return(false, errors.New("connection refused"))

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
		mockUserStore.On("UnRegisterSession", c.Request().Context(), "test-session")

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
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=newuser&password=secret&confirmPassword=secret", "")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, nil)
		mockUserStore.On("RegisterUser", c.Request().Context(), "newuser", "secret").Return(userUuid)
		mockUserStore.On("RegisterSession", c.Request().Context(), userUuid, mock.AnythingOfType("string"))

		h := &Handler{
			UserStore: mockUserStore,
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
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=existing&password=secret&confirmPassword=secret", "")

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "existing").Return(true, nil)

		h := &Handler{
			UserStore: mockUserStore,
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Username Taken")
	})

	t.Run("passwords do not match", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=newuser&password=secret&confirmPassword=other", "")

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, nil)

		h := &Handler{
			UserStore: mockUserStore,
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Passwords Do Not Match")
	})

	t.Run("database error checking username", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=newuser&password=secret&confirmPassword=secret", "")

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, errors.New("connection refused"))

		h := &Handler{
			UserStore: mockUserStore,
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
