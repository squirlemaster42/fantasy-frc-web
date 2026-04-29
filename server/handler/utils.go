package handler

import (
	"bytes"
	"context"
	"server/middleware"
	"server/types"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func (h *Handler) Render(c echo.Context, component templ.Component) error {
	ctx := h.injectFaroData(c, c.Request().Context())
	return component.Render(ctx, c.Response())
}

func (h *Handler) RenderError(c echo.Context, status int, component templ.Component) error {
	var buf bytes.Buffer
	ctx := h.injectFaroData(c, c.Request().Context())
	err := component.Render(ctx, &buf)
	if err != nil {
		return err
	}
	return c.HTML(status, buf.String())
}

func (h *Handler) injectFaroData(c echo.Context, ctx context.Context) context.Context {
	nonce := middleware.GetNonce(c)
	sessionTok := ""
	userTok, _ := c.Cookie("sessionToken")
	if userTok != nil {
		sessionTok = userTok.Value
	}
	faroToken := h.generateFaroToken(sessionTok)
	return context.WithValue(ctx, types.FaroContextKey, types.FaroData{
		Token: faroToken,
		Nonce: nonce,
	})
}
