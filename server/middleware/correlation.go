package middleware

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type contextKey string

const correlationIDKey contextKey = "correlation_id"

func CorrelationID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			corrID := c.Request().Header.Get("X-Correlation-ID")
			if corrID == "" {
				corrID = uuid.New().String()
			}

			c.Response().Header().Set("X-Correlation-ID", corrID)

			ctx := context.WithValue(c.Request().Context(), correlationIDKey, corrID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

func GetCorrelationID(ctx context.Context) string {
	if corrID, ok := ctx.Value(correlationIDKey).(string); ok {
		return corrID
	}
	return ""
}

func LogWithContext(ctx context.Context) *slog.Logger {
	corrID := GetCorrelationID(ctx)
	if corrID != "" {
		return slog.With("correlation_id", corrID)
	}
	return slog.Default()
}
