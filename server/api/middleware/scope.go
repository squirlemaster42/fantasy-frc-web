package apimiddleware

import (
	"server/api"
	"slices"

	"github.com/labstack/echo/v5"
)

// fullAccessScope grants all machine API permissions.
const fullAccessScope = "full_access"

// RequireScopes returns middleware that requires the authenticated JWT to
// contain at least one of the provided scopes, or the full_access scope.
func RequireScopes(required ...Scope) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			scopes, ok := GetScopes(c)
			if !ok {
				api.Forbidden(c.Response(), "Missing scopes claim")
				return nil
			}

			if slices.Contains(scopes, fullAccessScope) {
				return next(c)
			}

			for _, r := range required {
				if slices.Contains(scopes, string(r)) {
					return next(c)
				}
			}

			api.Forbidden(c.Response(), "Insufficient scope")
			return nil
		}
	}
}
