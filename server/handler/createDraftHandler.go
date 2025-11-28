package handler

import (
	"fmt"
	"log/slog"
	"server/assert"
	"server/model"
	"server/view/draft"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewCreateDraft(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Handle View Create Draft")
    slog.Info("Got request to serve the create draft page")

    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")

    userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userUuid)

    draftCreateIndex := draft.DraftProfileIndex(c, model.DraftModel{ Id: -1 }, true)
    draftCreate := draft.DraftProfile(c, " | Create Draft", true, username, draftCreateIndex, -1)
    err = Render(c, draftCreate)
    assert.NoError(err, "Handle View Draft Create Failed To Render")
    return nil
}

func (h *Handler) HandleCreateDraftPost(c echo.Context) error {
    slog.Info("Got request to create a draft")
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

    userUuid := model.GetUserBySessionToken(h.Database, sessionToken.Value)
    username := model.GetUsername(h.Database, userUuid)

    draftModel := model.DraftModel {
        Owner: model.User{
            UserUuid: userUuid,
        },
        DisplayName: draftName,
        Description: description,
        Interval: intInterval,
        StartTime: parsedStartTime,
        EndTime: parsedEndTime,
        Status: model.FILLING,
    }

    slog.Info("Created Draft for user", "Draft", draftModel.String(), "User", username)

    draftId := model.CreateDraft(h.Database, &draftModel)
    slog.Info("Draft created. Redirecting to /u/draft/:draftId:/profile", "Draft Id", draftId)
    c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
    return nil
}
