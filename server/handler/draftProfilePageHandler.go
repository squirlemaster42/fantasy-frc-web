package handler

import (
	"server/assert"
	"server/model"
    draftView "server/view/draft"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewDraftProfile(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Handle update Draft Profile")
    draftId, err := strconv.Atoi(c.Param("id"))
    assert.NoError(err, "Failed to convert draft id to int")
    draft := model.GetDraft(h.Database, draftId)

    draftIndex := draftView.DraftCreateIndex(false, "")
    draftView := draftView.DraftCreate(" | Draft Profile", false, draftIndex)
    err = Render(c, draftView)
    return nil
}

func (h *Handler) HandleUpdateDraftProfile(c echo.Context) error {
    //TODO We need to update the draft settings
    return nil
}
