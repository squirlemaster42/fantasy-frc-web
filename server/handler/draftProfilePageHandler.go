package handler

import "github.com/labstack/echo/v4"

func (h *Handler) HandleViewDraftProfile(c echo.Context) error {
    c.Param("id")
    //TODO we need to load the current settings for the draft in the url and show those
    return nil
}

func (h *Handler) HandleUpdateDraftProfile(c echo.Context) error {
    c.Param("id")
    //TODO We need to update the draft settings
    return nil
}
