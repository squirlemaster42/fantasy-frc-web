package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"server/log"
)

func TestCorrelationID_UsesExistingHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(log.CorrelationIDHeader, "existing-correlation-id")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := CorrelationID()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, "existing-correlation-id", rec.Header().Get(log.CorrelationIDHeader))
	assert.Equal(t, "existing-correlation-id", log.GetCorrelationID(c.Request().Context()))
}

func TestCorrelationID_GeneratesNewIdWhenMissing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := CorrelationID()
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	generated := rec.Header().Get(log.CorrelationIDHeader)
	assert.NotEmpty(t, generated)
	assert.Equal(t, generated, log.GetCorrelationID(c.Request().Context()))
}
