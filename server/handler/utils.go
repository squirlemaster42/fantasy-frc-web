package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"server/middleware"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func Render(c echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response())
}

func RenderError(c echo.Context, status int, component templ.Component) error {
	var buf bytes.Buffer
	err := component.Render(c.Request().Context(), &buf)
	if err != nil {
		return err
	}
	return c.HTML(status, buf.String())
}

func RenderToString(ctx context.Context, component templ.Component) (string, error) {
	var buf bytes.Buffer
	err := component.Render(ctx, &buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// generateCSRFCookie creates a double-submit CSRF cookie for unauthenticated forms
// (login/register). It returns the token to embed in the form.
func generateCSRFCookie(c echo.Context) (string, error) {
	// Check if cookie already exists
	existing, err := c.Cookie(middleware.CsrfCookieName)
	if err == nil && existing.Value != "" {
		return existing.Value, nil
	}

	// Generate new random token
	b := make([]byte, middleware.CsrfTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	cookie := new(http.Cookie)
	cookie.Name = middleware.CsrfCookieName
	cookie.Value = token
	cookie.Path = "/"
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Secure = c.Scheme() == "https"
	cookie.HttpOnly = false
	c.SetCookie(cookie)

	return token, nil
}

// validateCSRFCookie checks the double-submit CSRF token for unauthenticated forms.
func validateCSRFCookie(c echo.Context) bool {
	submitted := c.FormValue(middleware.CsrfTokenFieldName)
	if submitted == "" {
		return false
	}
	cookie, err := c.Cookie(middleware.CsrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(submitted), []byte(cookie.Value)) == 1
}
