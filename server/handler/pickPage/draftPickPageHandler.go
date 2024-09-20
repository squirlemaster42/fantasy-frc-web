package handler

import (
	"server/model"

	"github.com/labstack/echo/v4"
)

func (h *Handler) MakeNextPick(c echo.Context, pick model.Team) error {
    sessionToken, err := c.Cookie("sessionToken")
    userId :=  model.GetUserBySessionToken(h.Database, sessionToken.Value)

    if err != nil {
        //User not logged in
    }

    return nil
}
