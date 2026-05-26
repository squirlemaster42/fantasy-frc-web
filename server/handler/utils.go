package handler

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

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

// generateCSRFCookie creates a double-submit CSRF cookie for unauthenticated forms
// (login/register). It returns the token to embed in the form.
func generateCSRFCookie(c echo.Context) (string, error) {
	// Check if cookie already exists
	existing, err := c.Cookie("csrf_cookie")
	if err == nil && existing.Value != "" {
		return existing.Value, nil
	}

	// Generate new random token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	cookie := new(http.Cookie)
	cookie.Name = "csrf_cookie"
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
	submitted := c.FormValue("csrf_token")
	if submitted == "" {
		return false
	}
	cookie, err := c.Cookie("csrf_cookie")
	if err != nil || cookie.Value == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(submitted), []byte(cookie.Value)) == 1
}
