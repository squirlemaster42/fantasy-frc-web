package apimiddleware

import (
	"net/http"
	"net/http/httptest"
	"server/authentication/jwt"
	"testing"
	"time"
	"uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTAuth_Success(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")
	userUuid := uuid.New()
	token, err := jwt.Sign(userUuid, key, 15*time.Minute, nil)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/drafts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := JWTAuth(key)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	extracted, ok := GetUserUuid(c)
	assert.True(t, ok)
	assert.Equal(t, userUuid, extracted)
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")

	e := echo.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/drafts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := JWTAuth(key)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")

	e := echo.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/drafts", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := JWTAuth(key)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")
	userUuid := uuid.New()
	token, err := jwt.Sign(userUuid, key, -1*time.Second, nil)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/drafts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := JWTAuth(key)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_WrongScheme(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")
	userUuid := uuid.New()
	token, err := jwt.Sign(userUuid, key, 15*time.Minute, nil)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/drafts", nil)
	req.Header.Set("Authorization", "Basic "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := JWTAuth(key)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
