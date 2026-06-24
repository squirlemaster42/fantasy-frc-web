package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net/http"
	"server/log"

	"github.com/labstack/echo/v4"
)

type csrfContextKey string

const CsrfTokenKey csrfContextKey = "csrfToken"

type CSRFMiddleware struct {
	secret       []byte
	secureCookie bool
}

func NewCSRF(secret string, secureCookie bool) *CSRFMiddleware {
	if secret == "" {
		panic("CSRF_SECRET environment variable not set")
	}
	return &CSRFMiddleware{
		secret:       []byte(secret),
		secureCookie: secureCookie,
	}
}

// GenerateToken creates a CSRF token derived from the session token.
func (c *CSRFMiddleware) GenerateToken(sessionToken string) string {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(sessionToken))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// CSRF returns middleware that validates CSRF tokens on state-changing requests
// and stores the expected token in the Echo context for handlers to use.
func (c *CSRFMiddleware) CSRF() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			method := ctx.Request().Method
			path := ctx.Request().URL.Path

			// Skip for login/register (they use double-submit cookie)
			if path == "/login" || path == "/register" {
				return next(ctx)
			}

			// Get expected token from session and store it in context
			sessionCookie, err := ctx.Cookie("sessionToken")
			if err == nil && sessionCookie.Value != "" {
				expectedToken := c.GenerateToken(sessionCookie.Value)
				ctx.Set(string(CsrfTokenKey), expectedToken)

				// Set non-HttpOnly cookie so JS can read it for HTMX requests
				csrfCookie := new(http.Cookie)
				csrfCookie.Name = "csrf_token"
				csrfCookie.Value = expectedToken
				csrfCookie.Path = "/"
				csrfCookie.SameSite = http.SameSiteLaxMode
				csrfCookie.Secure = c.secureCookie
				csrfCookie.HttpOnly = false
				ctx.SetCookie(csrfCookie)
			}

			// Skip validation for safe methods
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				return next(ctx)
			}

		// Validate token for state-changing requests
		if err != nil || sessionCookie.Value == "" {
			log.Warn(ctx.Request().Context(), "CSRF validation failed", "path", path, "method", method, "ip", ctx.RealIP(), "reason", "no_session_token")
			return ctx.NoContent(http.StatusForbidden)
		}

		// Get token from form or header
		var submittedToken string
		if ctx.Request().FormValue("csrf_token") != "" {
			submittedToken = ctx.Request().FormValue("csrf_token")
		} else {
			submittedToken = ctx.Request().Header.Get("X-CSRF-Token")
		}

		expectedToken := c.GenerateToken(sessionCookie.Value)
		if subtle.ConstantTimeCompare([]byte(submittedToken), []byte(expectedToken)) != 1 {
			log.Warn(ctx.Request().Context(), "CSRF validation failed", "path", path, "method", method, "ip", ctx.RealIP(), "reason", "token_mismatch")
			return ctx.NoContent(http.StatusForbidden)
		}

			return next(ctx)
		}
	}
}
