package middleware

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "route", "status", "status_class"},
	)
)

func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			status := c.Response().Status
			method := c.Request().Method
			route := c.Path()
			if route == "" {
				route = "unknown"
			}

			requestCount.WithLabelValues(
				method,
				route,
				strconv.Itoa(status),
				strconv.Itoa(status / 100),
			).Inc()

			return err
		}
	}
}
