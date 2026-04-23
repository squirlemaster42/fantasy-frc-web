package handler

import (
	"errors"
	"fmt"
	"net/http"
	"server/assert"
	"server/log"
	"server/model"
	draftView "server/view/draft"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewDraftProfile(c echo.Context) error {
	log.Info(c.Request().Context(), "Got a request to serve the draft profile page")
	assert := assert.CreateAssertWithContext("Handle update Draft Profile")

	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id", "Draft Id String", c.Param("id"), "Error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}

	// TODO I think this should go through the draft manager
	draftModel, err := model.GetDraft(h.Database, draftId)
	if err != nil {
		//We want to redirect back to the home screen
		log.Warn(c.Request().Context(), "User attempted to visit incorrect draft id", "User Uuid", userUuid, "Draft Id", draftId, "Error", err)
		return c.Redirect(http.StatusSeeOther, "/u/home")
	}

	isOwner := userUuid == draftModel.Owner.UserUuid

	if draftModel.StartTime.IsZero() {
		log.Info(c.Request().Context(), "Found draft without a start time setting to an hour from now", "Draft Id", draftId, "Now", time.Now())
		draftModel.StartTime = time.Now().Add(1 * time.Hour)
	}

	if draftModel.EndTime.IsZero() {
		log.Info(c.Request().Context(), "Found draft without an end time setting to three days from now", "Draft Id", draftId, "Now", time.Now())
		draftModel.EndTime = time.Now().Add(72 * time.Hour)
	}

	draftIndex := draftView.DraftProfileIndex(draftModel, isOwner)
	draftView := draftView.DraftProfile(" | Draft Profile", true, username, draftIndex, draftId, isOwner)
	return Render(c, draftView)
}

func (h *Handler) HandleUpdateDraftProfile(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to update a draft")
	assert := assert.CreateAssertWithContext("Handle Update Draft Profile")

	draftId, err := strconv.Atoi(c.Param("id"))

	assert.NoError(err, "Could not parse draftId from params")
	assert.AddContext("Draft Id", draftId)

	draftName := c.FormValue("draftName")
	description := c.FormValue("description")
	interval := c.FormValue("interval")
	startTime := c.FormValue("startTime")
	endTime := c.FormValue("endTime")
	discordWebhook := c.FormValue("discordWebhook")

	sessionToken, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get session cookie")

	intInterval, err := strconv.Atoi(interval)
	assert.NoError(err, "Failed to parse interval")

	layout := "2006-01-02T15:04:05"
	parsedStartTime, err := time.Parse(layout, startTime)
	assert.NoError(err, "Failed to parse start time")
	parsedEndTime, err := time.Parse(layout, endTime)
	assert.NoError(err, "Failed to parse end time")

	userUuid := model.GetUserBySessionToken(h.Database, sessionToken.Value)

	draftModel, err := model.GetDraft(h.Database, draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to write to invalid draft id", "User Uuid", userUuid, "Draft Id", draftId)
		return nil
	}

	if draftModel.Owner.UserUuid != userUuid {
		//The user would need to hand craft this payload
		//so for now we just won't tell them what is wrong
		//because it is probably malicious
		log.Info(c.Request().Context(), "User tried to update draft but was not the owner.", "User Uuid", userUuid, "DraftId", draftId, "Owner Id", draftModel.Owner.UserUuid)
		return nil
	}

	draftModel = model.DraftModel{
		Id: draftId,
		Owner: model.User{
			UserUuid: userUuid,
		},
		DisplayName:    draftName,
		Description:    description,
		Interval:       intInterval,
		StartTime:      parsedStartTime,
		EndTime:        parsedEndTime,
		DiscordWebhook: discordWebhook,
	}

	err = h.DraftManager.UpdateDraft(draftModel)

	if err != nil {
		return err
	}

	log.Info(c.Request().Context(), "Draft updated, reloading page", "Draft Id", draftId)
	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
	return nil
}

func (h *Handler) SearchPlayers(c echo.Context) error {
	splitSource := strings.Split(c.Request().Header["Hx-Current-Url"][0], "/")
	draftId, err := strconv.Atoi(splitSource[len(splitSource)-2])
	assert.NoErrorCF(err, "Failed to parse draft Id")
	searchInput := c.FormValue("search")
	log.Debug(c.Request().Context(), "Got request to search users")

	users, err := model.SearchUsers(h.Database, searchInput, draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to search users", "Draft Id", draftId, "Search Input", searchInput, "Error", err)
		return nil
	}

	draftModel, err := model.GetDraft(h.Database, draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to search for players in an invalid draft", "Draft Id", draftId, "Error", err)
		return nil
	}

	userTok, err := c.Cookie("sessionToken")
	assert.NoErrorCF(err, "Failed to get user token")
	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)

	isOwner := userUuid == draftModel.Owner.UserUuid

	searchResults := draftView.PlayerSearchResults(users, draftId, isOwner)
	err = Render(c, searchResults)
	return err
}

func (h *Handler) InviteDraftPlayer(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Invite Draft Player")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")
	draftIdStr := c.Param("id")
	invitingUserUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	draftId, err := strconv.Atoi(draftIdStr)
	assert.NoError(err, "Invalid draft id")
	userUuidString := c.FormValue("userUuid")
	assert.AddContext("User UUID String", userUuidString)
	userUuid, err := uuid.Parse(userUuidString)
	assert.NoError(err, "Failed to parse user guid")

	// Check that the draft is in the correct state
	draft, err := h.DraftManager.GetDraft(draftId, false)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to load draft", "Draft Id", draftId, "Error", err)
		return err
	}

	if draft.GetStatus() != model.FILLING {
		return c.String(http.StatusBadRequest, "Draft must be in FILLING state to invite players")
	}

	_, err = model.InvitePlayer(h.Database, draftId, invitingUserUuid, userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to invite player", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to invite player")
	}

	assert.NoError(err, "Failed to parse draft Id")
	searchInput := c.FormValue("search")
	log.Debug(c.Request().Context(), "Got request to search users")

	users, err := model.SearchUsers(h.Database, searchInput, draftId)

	if err != nil {
		log.Warn(c.Request().Context(), "Failed to search users", "Draft Id", draftId, "Search Input", searchInput, "Error", err)
		return nil
	}

	draftModel, err := model.GetDraft(h.Database, draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to invite player to invalid draft", "Draft Id", draftId, "User Uuid", userUuid, "Error", err)
	}

	players := draftModel.Players

	isOwner := invitingUserUuid == draftModel.Owner.UserUuid

	updatedPage := draftView.UpdateAfterInvite(users, draftId, players, isOwner)
	err = Render(c, updatedPage)

	return err
}

func (h *Handler) HandleStartDraft(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Start Draft")
	userTok, err := c.Cookie("sessionToken")
	// Session token should always be here because the middleware should have
	// checked for it
	assert.NoError(err, "Failed to get user token.")
	draftIdStr := c.Param("id")
	log.Info(c.Request().Context(), "Got a request to start a draft", "Draft Id", draftIdStr)
	requestingUser := model.GetUserBySessionToken(h.Database, userTok.Value)
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Warn(c.Request().Context(), "Could not parse draftId", "Draft Id Str", draftIdStr, "Error", err)
		c.Response().Status = http.StatusBadRequest
		page := draftView.StartDraftButton(
			fmt.Sprintf("/u/draft/%d/startDraft", draftId),
			"Draft Id is not a number",
			false,
		)
		return Render(c, page)
	}

	draft, err := h.DraftManager.GetDraft(draftId, true)
	if err != nil {
		log.Warn(c.Request().Context(), "Could not load draft", "Draft Id", draftId, "Error", err)
		c.Response().Status = http.StatusBadRequest
		page := draftView.StartDraftButton(fmt.Sprintf("/u/draft/%d/startDraft", draftId), "Could not load draft", false)
		return Render(c, page)
	}

	if draft.GetOwner().UserUuid != requestingUser {
		log.Warn(c.Request().Context(), "User is not draft owner", "Draft Id", draftId, "User", requestingUser)
		c.Response().Status = http.StatusUnauthorized
		page := draftView.StartDraftButton(fmt.Sprintf("/u/draft/%d/startDraft", draftId), "Permission Denied", false)
		return Render(c, page)
	}

	// Check that eight players have accepted the draft
	numAccepted := 0
	fmt.Println(draft.Model.String())
	for _, p := range draft.Model.Players {
		if !p.Pending {
			numAccepted++
		}
	}

	if numAccepted != 8 {
		log.Warn(c.Request().Context(), "User attempted to start a draft with an incorrect number of players", "Draft Id", draftId, "Num Accepted", numAccepted)
		// TODO Show this error to users
		return errors.New("draft does not have the correct number of accepted players")
	}

	// Cancel the invites for players who have not accepted the draft
	model.CancelOutstandingInvites(h.Database, draftId)

	log.Info(c.Request().Context(), "Requesting draft state change to picking", "Draft Id", draftId)
	err = h.DraftManager.ExecuteDraftStateTransition(draftId, model.WAITING_TO_START)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		return err
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
	return nil
}
