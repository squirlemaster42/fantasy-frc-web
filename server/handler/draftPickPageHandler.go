package handler

import (
	"server/model"
	"server/view/draft"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) ServePickPage(c echo.Context) error {
    draftId, err := strconv.Atoi(c.Param("id"))
    draftModel := model.GetDraft(h.Database, draftId)
    pickPageIndex := draft.DraftPickIndex(draftModel)
    pickPageView := draft.DraftPick(" | " + draftModel.DisplayName, false, pickPageIndex)
    err = Render(c, pickPageView)

    return err
}
