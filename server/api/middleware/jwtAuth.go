package apimiddleware

import (
	"errors"
	"server/api"
	"server/authentication/jwt"
	"strings"
	"uuid"

	"github.com/labstack/echo/v5"
)

// JWTAuth returns Echo middleware that validates a Bearer JWT and stores the
// user UUID in the context under the key "userUuid".
func JWTAuth(signingKey []byte) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				api.Unauthorized(c.Response(), "Authorization header is required")
				return nil
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				api.Unauthorized(c.Response(), "Authorization header must be Bearer token")
				return nil
			}

			userUuid, scopes, err := jwt.Validate(parts[1], signingKey)
			if err != nil {
				jwt.LogValidationFailure(c.Request().Context(), err)
				if errors.Is(err, jwt.ErrExpiredToken) {
					api.Unauthorized(c.Response(), "Token expired")
				} else {
					api.Unauthorized(c.Response(), "Invalid token")
				}
				return nil
			}

			c.Set("userUuid", userUuid)
			c.Set("scopes", scopes)
			return next(c)
		}
	}
}

// GetUserUuid extracts the authenticated user UUID from the Echo context.
func GetUserUuid(c *echo.Context) (uuid.UUID, bool) {
	v := c.Get("userUuid")
	if v == nil {
		return uuid.UUID{}, false
	}
	userUuid, ok := v.(uuid.UUID)
	return userUuid, ok
}

// GetScopes extracts the authenticated scopes from the Echo context.
func GetScopes(c *echo.Context) ([]string, bool) {
	v := c.Get("scopes")
	if v == nil {
		return nil, false
	}
	scopes, ok := v.([]string)
	return scopes, ok
}
