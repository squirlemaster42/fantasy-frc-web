package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) GetTeamAvatar(c echo.Context) error {
	teamNum, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return errors.New("Id must be a valid team number")
	}

	// TODO cache base64 in database
	base64Str, err := h.TbaHandler.MakeTeamAvatarRequest(fmt.Sprintf("frc%d", teamNum))
	if err != nil {
		return err
	}

	image, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return errors.New("Failed to decode team image")
	}

	return c.Blob(http.StatusOK, "image/png", image)
}
