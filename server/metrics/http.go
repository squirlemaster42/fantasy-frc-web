package metrics

import (
	"context"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	httpRequestCount          metric.Int64Counter
	httpRequestDuration       metric.Float64Histogram
	httpRequestErrors         metric.Int64Counter
	authenticatedRequestCount metric.Int64Counter
)

func InitHTTPMetrics() {
	meter := otel.Meter("fantasy-frc-web")
	var err error
	httpRequestCount, err = meter.Int64Counter(
		"http.request.count",
		metric.WithDescription("Total HTTP requests"),
	)
	if err != nil {
		panic("failed to create http.request.count: " + err.Error())
	}

	httpRequestDuration, err = meter.Float64Histogram(
		"http.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic("failed to create http.request.duration: " + err.Error())
	}

	httpRequestErrors, err = meter.Int64Counter(
		"http.request.errors",
		metric.WithDescription("Total HTTP request errors"),
	)
	if err != nil {
		panic("failed to create http.request.errors: " + err.Error())
	}

	authenticatedRequestCount, err = meter.Int64Counter(
		"http.authenticated.request.count",
		metric.WithDescription("Total authenticated HTTP requests by route and method"),
	)
	if err != nil {
		panic("failed to create http.authenticated.request.count: " + err.Error())
	}
}

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

			attrs := []attribute.KeyValue{
				attribute.String("http.method", method),
				attribute.String("http.route", route),
				attribute.String("http.status_code", strconv.Itoa(status)),
				attribute.String("http.status_class", statusClass),
			}

			if httpRequestCount != nil {
				httpRequestCount.Add(c.Request().Context(), 1, metric.WithAttributes(attrs...))
			}
			if httpRequestDuration != nil {
				httpRequestDuration.Record(c.Request().Context(), duration,
					metric.WithAttributes(
						attribute.String("http.method", method),
						attribute.String("http.route", route),
						attribute.String("http.status_class", statusClass),
					),
				)
			}
			if status >= 400 && httpRequestErrors != nil {
				httpRequestErrors.Add(c.Request().Context(), 1, metric.WithAttributes(attrs...))
			}

			return err
		}
	}
}

func RecordAuthenticatedRequest(method, route string) {
	if authenticatedRequestCount == nil {
		return
	}
	if route == "" {
		route = "unknown"
	}
	authenticatedRequestCount.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.route", route),
		),
	)
}
