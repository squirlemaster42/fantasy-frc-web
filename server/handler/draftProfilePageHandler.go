package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"server/assert"
	"server/model"
	draftView "server/view/draft"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewDraftProfile(c echo.Context) error {
	slog.Info("Got a request to serve the draft profile page")
	assert := assert.CreateAssertWithContext("Handle update Draft Profile")

	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userId := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userId)

	draftId, err := strconv.Atoi(c.Param("id"))
	assert.NoError(err, "Failed to convert draft id to int")
	draftModel, err := model.GetDraft(h.Database, draftId)
    if err != nil {
        //We want to redirect back to the home screen
        slog.Warn("User attempted to visit incorrect draft id", "User", userId, "Draft Id", draftId)
        c.Redirect(http.StatusSeeOther, "/u/home")
        return nil
    }

    isOwner := userId == draftModel.Owner.Id

	draftIndex := draftView.DraftProfileIndex(draftModel, isOwner)
	draftView := draftView.DraftProfile(" | Draft Profile", true, username, draftIndex, draftId)
	err = Render(c, draftView)
	return nil
}

func (h *Handler) HandleUpdateDraftProfile(c echo.Context) error {
    slog.Info("Got request to update a draft")
    assert := assert.CreateAssertWithContext("Handle Update Draft Profile")

    draftId, err := strconv.Atoi(c.Param("id"))

    assert.NoError(err, "Could not parse draftId from params")
    assert.AddContext("Draft Id", draftId)

    draftName := c.FormValue("draftName")
    description := c.FormValue("description")
    interval := c.FormValue("interval")
    startTime := c.FormValue("startTime")
    endTime := c.FormValue("endTime")
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

    draftModel, err := model.GetDraft(h.Database, draftId)
    if err != nil {
        slog.Warn("User attempted to write to invalid draft id", "User", userId, "Draft Id", draftId)
        return nil
    }

    if draftModel.Owner.Id != userId {
        //The user would need to hand craft this payload
        //so for now we just won't tell them what is wrong
        //because it is probably malicious
        slog.Info("User tried to update draft but was not the owner.", "UserId", userId, "DraftId", draftId, "Owner Id", draftModel.Owner.Id)
        return nil
    }

    draftModel = model.DraftModel {
        Id: draftId,
        Owner: model.User{Id: userId},
        DisplayName: draftName,
        Description: description,
        Interval: intInterval,
        StartTime: parsedStartTime,
        EndTime: parsedEndTime,
    }

    model.UpdateDraft(h.Database, &draftModel)

    slog.Info("Draft updated, reloading page", "Draft Id", draftId)
    c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
    return nil
}

func (h *Handler) SearchPlayers(c echo.Context) error {
	splitSource := strings.Split(c.Request().Header["Hx-Current-Url"][0], "/")
	draftId, err := strconv.Atoi(splitSource[len(splitSource)-2])
	assert.NoErrorCF(err, "Failed to parse draft Id")
	searchInput := c.FormValue("search")
	slog.Info("Got request to search users")

	users := model.SearchUsers(h.Database, searchInput, draftId)

    draftModel, err :=  model.GetDraft(h.Database, draftId)
    if err != nil {
        slog.Warn("User attempted to search for players in an invalid draft", "Draft Id", draftId)
        return nil
    }

    userTok, err := c.Cookie("sessionToken")
	assert.NoErrorCF(err, "Failed to get user token")
	userId := model.GetUserBySessionToken(h.Database, userTok.Value)

    isOwner := userId == draftModel.Owner.Id

	searchResults := draftView.PlayerSearchResults(users, draftId, isOwner)
	err = Render(c, searchResults)
	return err
}

func (h *Handler) InviteDraftPlayer(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Invite Draft Player")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")
	draftIdStr := c.Param("id")
	invitingPlayer := model.GetUserBySessionToken(h.Database, userTok.Value)
	draftId, err := strconv.Atoi(draftIdStr)
	assert.NoError(err, "Invalid draft id")
	userIdStr := c.FormValue("userId")
	userId, err := strconv.Atoi(userIdStr)
	assert.NoError(err, "Failed to parse user id")

	model.InvitePlayer(h.Database, draftId, invitingPlayer, userId)

	assert.NoError(err, "Failed to parse draft Id")
	searchInput := c.FormValue("search")
	slog.Info("Got request to search users")

	users := model.SearchUsers(h.Database, searchInput, draftId)

    draftModel, err := model.GetDraft(h.Database, draftId)
    if err != nil {
        slog.Warn("User attempted to invite player to invalid draft", "Draft Id", draftId, "User", userId)
    }

    players := draftModel.Players

    isOwner := invitingPlayer == draftModel.Owner.Id

	updatedPage := draftView.UpdateAfterInvite(users, draftId, players, isOwner)
	err = Render(c, updatedPage)

	return err
}
