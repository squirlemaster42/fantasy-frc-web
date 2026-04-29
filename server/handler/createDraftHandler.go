package handler

import (
	"fmt"
	"net/http"
	"server/assert"
	"server/log"
	"server/model"
	"server/view/draft"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewCreateDraft(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle View Create Draft")
	log.Info(c.Request().Context(), "Got request to serve the create draft page")

	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)
	draftModel := model.DraftModel{
		Id: -1,
	}
	draftModel.StartTime = time.Now().Add(72 * time.Hour)
	draftModel.EndTime = time.Now().Add(144 * time.Hour)

	draftCreateIndex := draft.DraftProfileIndex(draftModel, true)
	draftCreate := draft.DraftProfile(" | Create Draft", true, username, draftCreateIndex, -1, true)
	err = h.Render(c, draftCreate)
	assert.NoError(err, "Handle View Draft Create Failed To Render")
	return nil
}

func (h *Handler) HandleCreateDraftPost(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to create a draft")
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

	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Interval must be a number, was %s", interval))
	}

	layout := "2006-01-02T15:04:05"
	parsedStartTime, err := time.Parse(layout, startTime)
	assert.NoError(err, "Failed to parse start time")
	parsedEndTime, err := time.Parse(layout, endTime)
	assert.NoError(err, "Failed to parse end time")

	userUuid := model.GetUserBySessionToken(h.Database, sessionToken.Value)
	username := model.GetUsername(h.Database, userUuid)

	draftModel := model.DraftModel{
		Owner: model.User{
			UserUuid: userUuid,
		},
		DisplayName: draftName,
		Description: description,
		Interval:    intInterval,
		StartTime:   parsedStartTime,
		EndTime:     parsedEndTime,
		Status:      model.FILLING,
	}

	log.Info(c.Request().Context(), "Created Draft for user", "User", username)

	draftId, err := model.CreateDraft(h.Database, &draftModel)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to create draft", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create draft")
	}
	log.Info(c.Request().Context(), "Draft created. Redirecting to /u/draft/:draftId:/profile", "Draft Id", draftId)
	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
	return nil
}
