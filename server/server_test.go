package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"server/assets"
	"server/authentication"
	"server/handler"
	modelmocks "server/model/mocks"
)

func setupTestServerConfig(t *testing.T) ServerConfig {
	t.Helper()

	return ServerConfig{
		ServerPort:       "8080",
		Handler:          handler.Handler{},
		MetricSecret:     "test-metric-secret",
		CsrfSecret:       "test-csrf-secret-32-bytes-long!!",
		RedisAddr:        "",
		RedisPassword:    "",
		RedisRateLimitDB: 1,
		PostsPerMinute:   100,
		RateLimitEnabled: false,
		TrustProxy:       false,
		AllowedOrigin:    "",
	}
}

func TestNewHTTPErrorHandler(t *testing.T) {
	t.Run("404 renders not found page", func(t *testing.T) {
		cfg := setupTestServerConfig(t)
		e := echo.New()
		e.HTTPErrorHandler = newHTTPErrorHandler(cfg)

		req := httptest.NewRequest(http.MethodGet, "/missing", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		e.HTTPErrorHandler(echo.NewHTTPError(http.StatusNotFound), c)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "404")
		assert.Contains(t, body, "Page not found")
		assert.Contains(t, body, `href="/"`)
	})

	t.Run("403 renders forbidden page", func(t *testing.T) {
		cfg := setupTestServerConfig(t)
		e := echo.New()
		e.HTTPErrorHandler = newHTTPErrorHandler(cfg)

		req := httptest.NewRequest(http.MethodGet, "/secret", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		e.HTTPErrorHandler(echo.NewHTTPError(http.StatusForbidden, "nope"), c)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "403")
		assert.Contains(t, body, "Access denied")
	})

	t.Run("500 renders server error page", func(t *testing.T) {
		cfg := setupTestServerConfig(t)
		e := echo.New()
		e.HTTPErrorHandler = newHTTPErrorHandler(cfg)

		req := httptest.NewRequest(http.MethodGet, "/boom", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		e.HTTPErrorHandler(errors.New("database exploded"), c)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "500")
		assert.Contains(t, body, "Server error")
	})

	t.Run("non standard status falls back to generic error page", func(t *testing.T) {
		cfg := setupTestServerConfig(t)
		e := echo.New()
		e.HTTPErrorHandler = newHTTPErrorHandler(cfg)

		req := httptest.NewRequest(http.MethodGet, "/teapot", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		e.HTTPErrorHandler(echo.NewHTTPError(http.StatusTeapot, "i'm a teapot"), c)

		assert.Equal(t, http.StatusTeapot, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "500")
		assert.Contains(t, body, "Server error")
	})

	t.Run("authenticated context renders protected layout", func(t *testing.T) {
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

		mockUserStore := modelmocks.NewMockUserStore(t)
		mockUserStore.On("GetUsername", mock.Anything, userUuid).Return("testuser", nil)

		cfg := setupTestServerConfig(t)
		cfg.Handler.Stores.UserStore = mockUserStore

		e := echo.New()
		e.HTTPErrorHandler = newHTTPErrorHandler(cfg)

		req := httptest.NewRequest(http.MethodGet, "/missing", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(string(authentication.UserUuidKey), userUuid)

		e.HTTPErrorHandler(echo.NewHTTPError(http.StatusNotFound), c)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "testuser")
		assert.Contains(t, body, `href="/u/home"`)
	})

	t.Run("committed response is not overwritten", func(t *testing.T) {
		cfg := setupTestServerConfig(t)
		e := echo.New()
		e.HTTPErrorHandler = newHTTPErrorHandler(cfg)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Response().WriteHeader(http.StatusOK)
		c.Response().Write([]byte("already written")) //nolint:errcheck

		e.HTTPErrorHandler(echo.NewHTTPError(http.StatusInternalServerError), c)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "already written")
	})
}

func TestRegisterSystemRoutes(t *testing.T) {
	cfg := setupTestServerConfig(t)
	e := echo.New()
	metricAuth := authentication.NewMetricAuth(cfg.MetricSecret)

	registerSystemRoutes(e, cfg, metricAuth)

	t.Run("healthz returns ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("metrics requires bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("metrics accepts valid bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("Authorization", "Bearer "+cfg.MetricSecret)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "go_gc_duration_seconds")
	})
}

func TestRegisterCatchAll(t *testing.T) {
	cfg := setupTestServerConfig(t)
	e := echo.New()
	e.HTTPErrorHandler = newHTTPErrorHandler(cfg)

	registerCatchAll(e)

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "404")
}

func TestStaticAssetRoutes(t *testing.T) {
	e := echo.New()

	e.Add(http.MethodGet, "/css/*", echo.StaticDirectoryHandler(assets.CSS(), false), cacheControlMiddleware)
	e.Add(http.MethodGet, "/img/*", echo.StaticDirectoryHandler(assets.Img(), false), cacheControlMiddleware)
	e.Add(http.MethodGet, "/js/*", echo.StaticDirectoryHandler(assets.JS(), false), cacheControlMiddleware)

	t.Run("css file has cache control", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/css/styles.css", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Cache-Control"), "public, max-age=")
	})

	t.Run("missing static asset returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/css/does-not-exist.css", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestCacheControlMiddleware(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/css/styles.css", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := cacheControlMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Contains(t, rec.Header().Get("Cache-Control"), "public, max-age=")
}
