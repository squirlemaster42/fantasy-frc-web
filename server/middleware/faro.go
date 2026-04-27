package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func FaroCORS(allowedOrigins []string) echo.MiddlewareFunc {
	return echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{http.MethodPost, http.MethodOptions},
		AllowHeaders:     []string{"Content-Type", "X-Faro-Token"},
		MaxAge:           86400,
	})
}

func CSPNonce() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			nonceBytes := make([]byte, 16)
			_, err := rand.Read(nonceBytes)
			if err != nil {
				return next(c)
			}
			nonce := base64.StdEncoding.EncodeToString(nonceBytes)

			c.Set("csp_nonce", nonce)

			origin := c.Request().Header.Get("Origin")
			allowedOrigins := []string{"'self'"}
			if origin != "" {
				allowedOrigins = append(allowedOrigins, origin)
			}

			csp := strings.Join([]string{
				"default-src 'self'",
				"script-src 'self' unpkg.com cdn.jsdelivr.net 'nonce-" + nonce + "'",
				"style-src 'self' 'unsafe-inline'",
				"img-src 'self' data:",
				"connect-src 'self'",
				"frame-src 'none'",
			}, "; ")

			c.Response().Header().Set("Content-Security-Policy", csp)

			return next(c)
		}
	}
}

func GetNonce(c echo.Context) string {
	nonce, _ := c.Get("csp_nonce").(string)
	return nonce
}
