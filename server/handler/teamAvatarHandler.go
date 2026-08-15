package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) GetTeamAvatar(c echo.Context) error {
	teamNum, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return errors.New("id must be a valid team number")
	}

	avatar, err := h.Services.AvatarStore.GetAvatar(c.Request().Context(), teamNum)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Cache-Control", fmt.Sprintf("private, max-age=%d", AvatarHttpCacheMaxAgeSeconds()))
	return c.Blob(http.StatusOK, "image/png", avatar)
}
