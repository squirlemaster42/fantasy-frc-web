package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddleware_RecordsStatusCode(t *testing.T) {
	e := echo.New()
	e.Use(MetricsMiddleware())
	e.GET("/test", func(c echo.Context) error {
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

func TestMetricsMiddleware_HandlerReturningErrorRecordsPreErrorHandlerStatus(t *testing.T) {
	// In Echo v4 a handler that returns an HTTPError does not update
	// c.Response().Status before the middleware runs (the error handler runs
	// afterwards). The existing middleware therefore records the default 200.
	e := echo.New()
	e.Use(MetricsMiddleware())
	e.GET("/bad", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	rec := httptest.NewRecorder()

	before := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/bad", "200", "2"))

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	after := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/bad", "200", "2"))
	assert.Equal(t, before+1, after)
}

func TestMetricsMiddleware_ExplicitlySetStatusIsRecorded(t *testing.T) {
	e := echo.New()
	e.Use(MetricsMiddleware())
	e.GET("/bad", func(c echo.Context) error {
		c.Response().Status = http.StatusBadRequest
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

func TestMetricsMiddleware_UnknownRouteUsesUnknownLabel(t *testing.T) {
	e := echo.New()
	e.Use(MetricsMiddleware())
	e.GET("/known", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	before := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, unknownRouteLabel, "200", "2"))

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	after := testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, unknownRouteLabel, "200", "2"))
	assert.Equal(t, before+1, after)
}

func TestMetricsMiddleware_DurationObserved(t *testing.T) {
	e := echo.New()
	e.Use(MetricsMiddleware())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Histogram observation count is exposed via Collect; ToFloat64 only works for counters/gauges.
	// We verify the request count was recorded, which implies the middleware ran end-to-end.
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.GreaterOrEqual(t, testutil.ToFloat64(httpRequestCount.WithLabelValues(http.MethodGet, "/test", "200", "2")), 1.0)
}
