package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v5"
	"server/log"
)

func Render(c *echo.Context, component templ.Component) error {
	if err := component.Render(c.Request().Context(), c.Response()); err != nil {
		return fmt.Errorf("failed to render component: %w", err)
	}
	return nil
}

func RenderError(c *echo.Context, status int, component templ.Component) error {
	var buf bytes.Buffer
	err := component.Render(c.Request().Context(), &buf)
	if err != nil {
		return fmt.Errorf("failed to render RenderError: %w", err)
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
		return "", fmt.Errorf("failed to render RenderToString: %w", err)
	}
	return buf.String(), nil
}

// RenderOrLog renders the component and logs a structured error on failure.
// It returns the render error unchanged so callers can still return it.
func RenderOrLog(c *echo.Context, component templ.Component, logMsg string, attrs ...any) error {
	if err := Render(c, component); err != nil {
		log.Error(c.Request().Context(), logMsg, append(attrs, "error", err)...)
		return err
	}
	return nil
}

// Redirect sends an HTTP redirect to the given path with SeeOther.
func Redirect(c *echo.Context, path string) error {
	return c.Redirect(http.StatusSeeOther, path)
}

// HXRedirect sets the HX-Redirect header and returns nil.
func HXRedirect(c *echo.Context, path string) error {
	c.Response().Header().Set("HX-Redirect", path)
	return nil
}

// BadRequest logs a warning, writes a 400 response, and returns nil. It is
// intended to be used as a terminal handler return.
func BadRequest(c *echo.Context, logMsg string, userMsg string, attrs ...any) error {
	log.Warn(c.Request().Context(), logMsg, append(attrs, "userMsg", userMsg)...)
	_ = c.String(http.StatusBadRequest, userMsg)
	return nil
}

// Forbidden logs a warning, writes a 403 response, and returns nil. It is
// intended to be used as a terminal handler return.
func Forbidden(c *echo.Context, logMsg string, userMsg string, attrs ...any) error {
	log.Warn(c.Request().Context(), logMsg, append(attrs, "userMsg", userMsg)...)
	_ = c.String(http.StatusForbidden, userMsg)
	return nil
}

// Unauthorized logs a warning, writes a 401 response, and returns nil. It is
// intended to be used as a terminal handler return.
func Unauthorized(c *echo.Context, logMsg string, userMsg string, attrs ...any) error {
	log.Warn(c.Request().Context(), logMsg, append(attrs, "userMsg", userMsg)...)
	_ = c.String(http.StatusUnauthorized, userMsg)
	return nil
}

// NotFound logs a warning, writes a 404 response, and returns nil. It is
// intended to be used as a terminal handler return.
func NotFound(c *echo.Context, logMsg string, userMsg string, attrs ...any) error {
	log.Warn(c.Request().Context(), logMsg, append(attrs, "userMsg", userMsg)...)
	_ = c.String(http.StatusNotFound, userMsg)
	return nil
}

// InternalServerError logs an error, writes a generic 500 response, and returns
// nil. It is intended to be used as a terminal handler return.
func InternalServerError(c *echo.Context, logMsg string, err error, attrs ...any) error {
	log.Error(c.Request().Context(), logMsg, append(attrs, "error", err)...)
	_ = c.String(http.StatusInternalServerError, "An error occurred")
	return nil
}

