package handler

import (
	"fmt"
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

func (h *Handler) HandleCreateDraftPost(c echo.Context) error {
    draftName := c.FormValue("draftName")
    description := c.FormValue("description")
    interval := c.FormValue("interval")
    startTime := c.FormValue("startTime")
    endTime := c.FormValue("endTime")

    fmt.Println(draftName)
    fmt.Println(description)
    fmt.Println(interval)
    fmt.Println(startTime)
    fmt.Println(endTime)

    draftCreateIndex := draft.DraftCreateIndex(false, "")
    draftCreate := draft.DraftCreate(" | Create Draft", false, draftCreateIndex)
    //TODO We should probably make tailwind work offline to make the dev experience better
    err := Render(c, draftCreate)
    assert.NoErrorCF(err, "Handle View Draft Create Failed To Render")
    return nil
}
