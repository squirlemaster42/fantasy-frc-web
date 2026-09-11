package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Use(MetricsMiddleware())
	e.HTTPErrorHandler = WrapHTTPErrorHandler(e.HTTPErrorHandler)
	return e
}

func TestMetricsMiddleware_RecordsSuccessfulRequest(t *testing.T) {
	e := newTestEcho()
	e.GET("/test", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	before := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/test", "200", "2"))

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	after := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/test", "200", "2"))
	assert.Equal(t, before+1, after)
}

func TestMetricsMiddleware_RecordsErrorStatus(t *testing.T) {
	e := newTestEcho()
	e.GET("/bad", func(c *echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	rec := httptest.NewRecorder()

	before := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/bad", "400", "4"))

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	after := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/bad", "400", "4"))
	assert.Equal(t, before+1, after)
}

func TestMetricsMiddleware_RecordsExplicitlySetStatus(t *testing.T) {
	e := newTestEcho()
	e.GET("/bad", func(c *echo.Context) error {
		if resp, err := echo.UnwrapResponse(c.Response()); err == nil {
			resp.Status = http.StatusBadRequest
		}
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	rec := httptest.NewRecorder()

	before := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/bad", "400", "4"))

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	after := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/bad", "400", "4"))
	assert.Equal(t, before+1, after)
}

func TestMetricsMiddleware_RecordsUnknownRouteAs404(t *testing.T) {
	e := newTestEcho()
	e.GET("/known", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	before := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, unknownRouteLabel, "404", "4"))

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	after := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, unknownRouteLabel, "404", "4"))
	assert.Equal(t, before+1, after)
}

func TestMetricsMiddleware_DurationObserved(t *testing.T) {
	e := newTestEcho()
	e.GET("/test", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.GreaterOrEqual(t, testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/test", "200", "2")), 1.0)
}

func TestWrapHTTPErrorHandler_DoesNotDoubleRecord(t *testing.T) {
	e := newTestEcho()
	e.GET("/ok", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()

	before := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/ok", "200", "2"))

	e.ServeHTTP(rec, req)

	after := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/ok", "200", "2"))
	assert.Equal(t, before+1, after)
}
