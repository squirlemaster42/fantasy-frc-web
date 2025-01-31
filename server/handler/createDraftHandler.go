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
    assert := assert.CreateAssertWithContext("Handle View Create Draft")
    h.Logger.Log("Got request to server the create draft page")

    userTok, err := c.Cookie("sessionToken")
    //TODO We should have already checked that the user has a token
    //here since they should not be able to access the page otherwise
    //There might be some sort of weird thing here where the middleware
    //validates the session token is good and then it expires a second later
    assert.NoError(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

    draftCreateIndex := draft.DraftProfileIndex(model.Draft{})
    draftCreate := draft.DraftProfile(" | Create Draft", true, username, draftCreateIndex)
    err = Render(c, draftCreate)
    assert.NoError(err, "Handle View Draft Create Failed To Render")
    return nil
}

func (h *Handler) HandleCreateDraftPost(c echo.Context) error {
    h.Logger.Log("Got request to create a draft")
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

    userId := model.GetUserBySessionToken(h.Database, sessionToken.Value)
    username := model.GetUsername(h.Database, userId)

    draftModel := model.Draft{
        Owner: model.User{Id: userId},
        DisplayName: draftName,
        Description: description,
        Interval: intInterval,
        StartTime: parsedStartTime,
        EndTime: parsedEndTime,
        Status: model.GetStatusString(model.FILLING),
    }

    h.Logger.Log(fmt.Sprintf("Created Draft: %s for user %s", draftModel.String(), username))

    draftId := model.CreateDraft(h.Database, &draftModel)
    err = c.Redirect(http.StatusSeeOther, fmt.Sprintf("/u/draft/%d/profile", draftId))
    assert.NoError(err, "Failed to redirect on successful draft creation")
    return nil
}
