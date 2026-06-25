package middleware

import (
	"server/log"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func CorrelationID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			corrID := c.Request().Header.Get(log.CorrelationIDHeader)
			if corrID == "" {
				corrID = uuid.New().String()
			}

			c.Response().Header().Set(log.CorrelationIDHeader, corrID)

			ctx := log.WithCorrelationID(c.Request().Context(), corrID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
