package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
)

type metricsContextKey string

const (
	metricsStartTimeKey metricsContextKey = "metrics_start_time"
	metricsRecordedKey  metricsContextKey = "metrics_recorded"
)

var (
	httpRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "route", "status", "status_class"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: httpRequestDurationBuckets,
		},
		[]string{"method", "route", "status_class"},
	)
	authenticatedRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authenticated_requests_total",
			Help: "Total authenticated HTTP requests by route and method",
		},
		[]string{"method", "route"},
	)
)

// MetricsMiddleware records successful requests and stores the start time for
// requests that fail and are later handled by WrapHTTPErrorHandler. This two-step
// design is needed because Echo runs the HTTP error handler after middleware has
// returned, so the final response status is not available inside the middleware
// for failing handlers.
func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ctx := context.WithValue(c.Request().Context(), metricsStartTimeKey, time.Now())
			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)
			if err == nil {
				recordHTTPRequestMetrics(c)
			}

			return err
		}
	}
}

// WrapHTTPErrorHandler wraps the Echo error handler to record metrics after the
// final response status has been determined. It only records if the request was
// not already recorded by MetricsMiddleware (i.e. successful requests).
func WrapHTTPErrorHandler(next echo.HTTPErrorHandler) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		next(c, err)

		if c.Request().Context().Value(metricsRecordedKey) == nil {
			recordHTTPRequestMetrics(c)
		}
	}
}

func recordHTTPRequestMetrics(c *echo.Context) {
	ctx := c.Request().Context()

	startVal := ctx.Value(metricsStartTimeKey)
	start, ok := startVal.(time.Time)
	if !ok {
		start = time.Now()
	}

	duration := time.Since(start).Seconds()
	status := http.StatusOK
	if resp, unwrapErr := echo.UnwrapResponse(c.Response()); unwrapErr == nil {
		status = resp.Status
	}
	method := c.Request().Method
	route := c.Path()
	if route == "" {
		route = unknownRouteLabel
	}
	statusClass := strconv.Itoa(status / 100)

	httpRequestCount.WithLabelValues(
		method,
		route,
		strconv.Itoa(status),
		statusClass,
	).Inc()

	httpRequestDuration.WithLabelValues(
		method,
		route,
		statusClass,
	).Observe(duration)

	c.SetRequest(c.Request().WithContext(context.WithValue(ctx, metricsRecordedKey, true)))
}

func RecordAuthenticatedRequest(method, route string) {
	if route == "" {
		route = "unknown"
	}
	authenticatedRequestCount.WithLabelValues(method, route).Inc()
}
