package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) GetTeamAvatar(c echo.Context) error {
	teamNum, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return errors.New("Id must be a valid team number")
	}

	avatar, err := h.AvatarStore.GetAvatar(teamNum)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Cache-Control", "private, max-age=604800")
	return c.Blob(http.StatusOK, "image/png", avatar)
}
