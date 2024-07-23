package handler

import (
	"server/assert"
	"server/model"
	"server/view/draft"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewCreateDraft(c echo.Context) error {
    draftCreateIndex := draft.DraftProfileIndex(model.Draft{})
    draftCreate := draft.DraftProfile(" | Create Draft", false, draftCreateIndex)
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
    sessionToken, err := c.Cookie("SessionToken")
    assert := assert.CreateAssertWithContext("Handle Create Draft Post")

    intInterval, err := strconv.Atoi(interval)
    assert.NoError(err, "Failed to parse interval")

    layout := "2006-01-02 15:04:05"
    parsedStartTime, err := time.Parse(layout, startTime)
    assert.NoError(err, "Failed to parse start time")
    parsedEndTime, err := time.Parse(layout, endTime)
    assert.NoError(err, "Failed to parse end time")

    draftModel := model.Draft{
        Owner: model.User{Id: model.GetUserBySessionToken(h.Database, sessionToken.Value)},
        DisplayName: draftName,
        Description: description,
        Interval: intInterval,
        StartTime: parsedStartTime,
        EndTime: parsedEndTime,
        Status: model.GetStatusString(model.FILLING),
    }

    //TODO we should probably grab the new draft id and then redirecet to that settings page
    model.CreateDraft(h.Database, &draftModel)

    draftCreateIndex := draft.DraftProfileIndex(model.Draft{})
    draftCreate := draft.DraftProfile(" | Create Draft", false, draftCreateIndex)
    //TODO We should probably make tailwind work offline to make the dev experience better
    err = Render(c, draftCreate)
    assert.NoError(err, "Handle View Draft Create Failed To Render")
    return nil
}
