package metrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
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
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"method", "route", "status_class"},
	)
)

func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start).Seconds()
			status := c.Response().Status
			method := c.Request().Method
			route := c.Path()
			if route == "" {
				route = "unknown"
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

			return err
		}
	}
}
