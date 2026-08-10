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
			Stores: StorageGroup{
				UserStore: mockUserStore,
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
		assert.NotEmpty(t, sessionCookie.Value)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=testuser&password=wrong&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "wrong").Return(false, nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
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

	t.Run("username not taken", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/login", "username=newuser&password=secret&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("ValidateLogin", c.Request().Context(), "newuser", "secret").Return(false, nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
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

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "secret").Return(false, errors.New("connection refused"))

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
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

		mockUserStore := mocks.NewMockUserStore(t)
		mockUserStore.On("UnRegisterSession", c.Request().Context(), "test-session").Return(nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
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
		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, nil)
		mockUserStore.On("RegisterUser", c.Request().Context(), "newuser", "Secret123").Return(userUuid, nil)
		mockUserStore.On("RegisterSession", c.Request().Context(), userUuid, mock.AnythingOfType("string")).Return(nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
			},
			Config: ConfigGroup{
				MinPasswordLength:           8,
				MinUsernameLength:           3,
				MaxUsernameLength:           32,
				UsernameAllowedSpecialChars: "_-",
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
		assert.NotEmpty(t, sessionCookie.Value)
	})

	t.Run("username taken", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=existing&password=Secret123&confirmPassword=Secret123&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "existing").Return(true, nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
			},
			Config: ConfigGroup{
				MinPasswordLength:           8,
				MinUsernameLength:           3,
				MaxUsernameLength:           32,
				UsernameAllowedSpecialChars: "_-",
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

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
			},
			Config: ConfigGroup{
				MinPasswordLength:           8,
				MinUsernameLength:           3,
				MaxUsernameLength:           32,
				UsernameAllowedSpecialChars: "_-",
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

		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("UsernameTaken", c.Request().Context(), "newuser").Return(false, errors.New("connection refused"))

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
			},
			Config: ConfigGroup{
				MinPasswordLength:           8,
				MinUsernameLength:           3,
				MaxUsernameLength:           32,
				UsernameAllowedSpecialChars: "_-",
			},
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Failed to check username availability")
	})

	t.Run("invalid username", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/register", "username=bad%20user&password=Secret123&confirmPassword=Secret123&csrf_token=test-csrf", "")
		c.Request().AddCookie(&http.Cookie{Name: "csrf_cookie", Value: "test-csrf"})

		mockUserStore := mocks.NewMockUserStore(t)

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
			},
			Config: ConfigGroup{
				MinPasswordLength:           8,
				MinUsernameLength:           3,
				MaxUsernameLength:           32,
				UsernameAllowedSpecialChars: "_-",
			},
		}

		err := h.HandlerRegisterPost(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Username cannot contain spaces")
	})
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		wantUser  string
		wantError string
	}{
		{"valid lowercase", "newuser", "newuser", ""},
		{"valid with hyphen", "new-user", "new-user", ""},
		{"valid with underscore", "new_user", "new_user", ""},
		{"valid mixed alphanumeric", "user123", "user123", ""},
		{"too short", "ab", "", "Username must be at least 3 characters"},
		{"too long", "thisusernameiswaytoolongtobeacceptable", "", "Username must be at most 32 characters"},
		{"leading space", " newuser", "", "Username cannot contain leading or trailing spaces"},
		{"trailing space", "newuser ", "", "Username cannot contain leading or trailing spaces"},
		{"internal space", "new user", "", "Username cannot contain spaces"},
		{"invalid character", "new@user", "", "Username can only contain letters, numbers, and _-"},
		{"empty", "", "", "Username must be at least 3 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser, gotError := validateUsername(tt.username, 3, 32, "_-")
			assert.Equal(t, tt.wantUser, gotUser)
			assert.Equal(t, tt.wantError, gotError)
		})
	}
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
