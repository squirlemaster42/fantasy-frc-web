package handler

import (
	"bytes"
	"context"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v5"
)

func Render(c *echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response())
}

func RenderError(c *echo.Context, status int, component templ.Component) error {
	var buf bytes.Buffer
	err := component.Render(c.Request().Context(), &buf)
	if err != nil {
		return err
	}
	return c.HTML(status, buf.String())
}

// setResponseStatus sets the Echo response status code. In Echo v5 the
// Response() method returns http.ResponseWriter, so the wrapped Echo response
// must be extracted before its Status field can be mutated.
func setResponseStatus(c *echo.Context, status int) {
	if resp, err := echo.UnwrapResponse(c.Response()); err == nil {
		resp.Status = status
	}
}

func RenderToString(ctx context.Context, component templ.Component) (string, error) {
	var buf bytes.Buffer
	err := component.Render(ctx, &buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

