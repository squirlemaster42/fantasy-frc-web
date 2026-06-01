package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func setupTestContext(t *testing.T, method string, target string, body string, cookieValue string) (*echo.Echo, echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return e, c, rec
}


