package handler

import (
	"fmt"
	"net/http"
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
    assert := assert.CreateAssertWithContext("Handle Create Draft Post")
    draftName := c.FormValue("draftName")
    description := c.FormValue("description")
    interval := c.FormValue("interval")
    startTime := c.FormValue("startTime")
    endTime := c.FormValue("endTime")
    //event := c.FormValue("event")
    sessionToken, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get session cookie")

    intInterval, err := strconv.Atoi(interval)
    assert.NoError(err, "Failed to parse interval")

    layout := "2006-01-02T15:04"
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

    draftId := model.CreateDraft(h.Database, &draftModel)
    err = c.Redirect(http.StatusSeeOther, fmt.Sprintf("/draft/%d/profile", draftId))
    assert.NoError(err, "Failed to redirect on successful draft creation")
    return nil
}
