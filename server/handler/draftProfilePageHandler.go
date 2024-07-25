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
    draftModel := model.GetDraft(h.Database, draftId)

    draftIndex := draftView.DraftProfileIndex(draftModel)
    draftView := draftView.DraftProfile(" | Draft Profile", false, draftIndex)
    err = Render(c, draftView)
    return nil
}

func (h *Handler) HandleUpdateDraftProfile(c echo.Context) error {
    //TODO We need to update the draft settings
    return nil
}
