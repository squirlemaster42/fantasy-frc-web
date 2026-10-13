package apimiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestRequireScopes_AllowsMatchingScope(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("scopes", []string{"drafts:read"})

	middleware := RequireScopes(ScopeDraftsRead)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireScopes_AllowsFullAccess(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("scopes", []string{"full_access"})

	middleware := RequireScopes(ScopeDraftsWrite)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireScopes_DeniesMissingScope(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("scopes", []string{"drafts:read"})

	middleware := RequireScopes(ScopeDraftsWrite)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireScopes_DeniesMissingClaim(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := RequireScopes(ScopeDraftsRead)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
