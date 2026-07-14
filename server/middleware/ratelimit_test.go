package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter_DisabledWhenAddrEmpty(t *testing.T) {
	limiter := NewRateLimiter("", "", 0)
	assert.NotNil(t, limiter)
	assert.Nil(t, limiter.client)
}

func TestNewRateLimiter_DisabledWhenRedisUnavailable(t *testing.T) {
	// Use a port that is very unlikely to be listening
	limiter := NewRateLimiter("localhost:1", "", 0)
	assert.NotNil(t, limiter)
	assert.Nil(t, limiter.client)
}

func TestNewRateLimiter_EnabledWithMiniredis(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	limiter := NewRateLimiter(s.Addr(), "", 0)
	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.client)
}

func TestRateLimiter_RateLimitLogin_BlocksAfterLimit(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	limiter := NewRateLimiter(s.Addr(), "", 0)
	middleware := limiter.RateLimitLogin()

	e := echo.New()

	// First 5 requests are allowed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Sixth request is blocked
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "900", rec.Header().Get("Retry-After"))
}

func TestRateLimiter_RateLimitRegister_BlocksAfterLimit(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	limiter := NewRateLimiter(s.Addr(), "", 0)
	middleware := limiter.RateLimitRegister()

	e := echo.New()

	// First 3 requests are allowed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/register", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Fourth request is blocked
	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimiter_RateLimitGeneral_SkipsSafeMethods(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	limiter := NewRateLimiter(s.Addr(), "", 0)
	middleware := limiter.RateLimitGeneral(1)

	e := echo.New()

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		req := httptest.NewRequest(method, "/u/home", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimiter_RateLimitGeneral_SkipsTbaWebhook(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	limiter := NewRateLimiter(s.Addr(), "", 0)
	middleware := limiter.RateLimitGeneral(1)

	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/tbaWebhook", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_RateLimitGeneral_BlocksAfterLimit(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	limiter := NewRateLimiter(s.Addr(), "", 0)
	middleware := limiter.RateLimitGeneral(1)

	e := echo.New()

	// First POST is allowed
	req := httptest.NewRequest(http.MethodPost, "/u/home", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second POST is blocked
	req = httptest.NewRequest(http.MethodPost, "/u/home", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	handler = middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
}

func TestRateLimiter_RateLimitGeneral_UsesUserUuidKey(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	limiter := NewRateLimiter(s.Addr(), "", 0)
	middleware := limiter.RateLimitGeneral(1)

	e := echo.New()
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// First POST by user is allowed
	req := httptest.NewRequest(http.MethodPost, "/u/home", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userUuid", userUuid)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second POST by same user is blocked
	req = httptest.NewRequest(http.MethodPost, "/u/home", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set("userUuid", userUuid)

	handler = middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimiter_RateLimitGeneral_DifferentUsersAreIndependent(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	limiter := NewRateLimiter(s.Addr(), "", 0)
	middleware := limiter.RateLimitGeneral(1)

	e := echo.New()

	user1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	user2 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	for _, userUuid := range []uuid.UUID{user1, user2} {
		req := httptest.NewRequest(http.MethodPost, "/u/home", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userUuid", userUuid)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimiter_RateLimitGeneral_FailOpenOnRedisError(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	limiter := NewRateLimiter(s.Addr(), "", 0)
	middleware := limiter.RateLimitGeneral(1)

	// Kill redis so subsequent requests fail open
	s.Close()

	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/u/home", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_checkLimit_AllowsWhenRedisNil(t *testing.T) {
	limiter := NewRateLimiter("", "", 0)

	allowed, _, err := limiter.checkLimit(t.Context(), "key", 1, time.Minute)
	assert.NoError(t, err)
	assert.True(t, allowed)
}
