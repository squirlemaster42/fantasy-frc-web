package handler

import (
	"server/assert"
	"server/view/draft"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewCreateDraft(c echo.Context) error {
    draftCreateIndex := draft.DraftCreateIndex(false, "")
    draftCreate := draft.DraftCreate(" | Create Draft", false, draftCreateIndex)
    //TODO We should probably make tailwind work offline to make the dev experience better
    err := Render(c, draftCreate)
    assert.NoErrorCF(err, "Handle View Draft Create Failed To Render")
    return nil
}
