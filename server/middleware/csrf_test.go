package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewCSRF_PanicsOnEmptySecret(t *testing.T) {
	assert.Panics(t, func() {
		NewCSRF("", false)
	})
}

func TestCSRF_GenerateToken(t *testing.T) {
	csrf := NewCSRF("test-secret", false)

	token1 := csrf.GenerateToken("session-token-1")
	token2 := csrf.GenerateToken("session-token-1")
	token3 := csrf.GenerateToken("session-token-2")

	assert.NotEmpty(t, token1)
	assert.Equal(t, token1, token2)
	assert.NotEqual(t, token1, token3)
}

func TestCSRF_SkipsLoginAndRegister(t *testing.T) {
	csrf := NewCSRF("test-secret", false)
	middleware := csrf.CSRF()

	for _, path := range []string{"/login", "/register"} {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestCSRF_SetsTokenCookieForSafeMethods(t *testing.T) {
	csrf := NewCSRF("test-secret", false)
	middleware := csrf.CSRF()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/some-page", nil)
	req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "session-123"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "csrf_token" {
			csrfCookie = cookie
			break
		}
	}
	assert.NotNil(t, csrfCookie)
	assert.Equal(t, csrf.GenerateToken("session-123"), csrfCookie.Value)
	assert.False(t, csrfCookie.HttpOnly)
	assert.Equal(t, "/", csrfCookie.Path)
}

func TestCSRF_SafeMethodsSkipValidation(t *testing.T) {
	csrf := NewCSRF("test-secret", false)
	middleware := csrf.CSRF()

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		e := echo.New()
		req := httptest.NewRequest(method, "/some-page", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestCSRF_RejectsMissingSessionToken(t *testing.T) {
	csrf := NewCSRF("test-secret", false)
	middleware := csrf.CSRF()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/some-page", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRF_RejectsInvalidTokenFromForm(t *testing.T) {
	csrf := NewCSRF("test-secret", false)
	middleware := csrf.CSRF()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/some-page", strings.NewReader("csrf_token=invalid-token"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "session-123"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRF_AcceptsValidTokenFromForm(t *testing.T) {
	csrf := NewCSRF("test-secret", false)
	middleware := csrf.CSRF()

	sessionToken := "session-123"
	validToken := csrf.GenerateToken(sessionToken)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/some-page", strings.NewReader("csrf_token="+validToken))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.AddCookie(&http.Cookie{Name: "sessionToken", Value: sessionToken})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRF_AcceptsValidTokenFromHeader(t *testing.T) {
	csrf := NewCSRF("test-secret", false)
	middleware := csrf.CSRF()

	sessionToken := "session-123"
	validToken := csrf.GenerateToken(sessionToken)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/some-page", nil)
	req.Header.Set("X-CSRF-Token", validToken)
	req.AddCookie(&http.Cookie{Name: "sessionToken", Value: sessionToken})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRF_StoresExpectedTokenInContext(t *testing.T) {
	csrf := NewCSRF("test-secret", false)
	middleware := csrf.CSRF()

	sessionToken := "session-123"

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/some-page", nil)
	req.AddCookie(&http.Cookie{Name: "sessionToken", Value: sessionToken})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var contextToken string
	handler := middleware(func(c echo.Context) error {
		contextToken = c.Get(string(CsrfTokenKey)).(string)
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, csrf.GenerateToken(sessionToken), contextToken)
}
