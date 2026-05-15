package handler

import (
	"fmt"
	"net/http"
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

	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id", "Draft Id String", c.Param("id"), "Error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}

	// TODO I think this should go through the draft manager
	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
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

	draftIndex := draftView.DraftProfileIndex(draftModel, isOwner, h.csrfToken(c))
	draftView := draftView.DraftProfile(" | Draft Profile", true, username, draftIndex, draftId, isOwner)
	return Render(c, draftView)
}

func (h *Handler) HandleUpdateDraftProfile(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to update a draft")

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Could not parse draftId from params", "Error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}

	draftName := c.FormValue("draftName")
	description := c.FormValue("description")
	interval := c.FormValue("interval")
	startTime := c.FormValue("startTime")
	endTime := c.FormValue("endTime")
	discordWebhook := c.FormValue("discordWebhook")

	intInterval, err := strconv.Atoi(interval)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse interval", "Error", err)
		return c.String(http.StatusBadRequest, "Invalid interval")
	}

	layout := "2006-01-02T15:04:05"
	parsedStartTime, err := time.Parse(layout, startTime)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse start time", "Error", err)
		return c.String(http.StatusBadRequest, "Invalid start time")
	}
	parsedEndTime, err := time.Parse(layout, endTime)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse end time", "Error", err)
		return c.String(http.StatusBadRequest, "Invalid end time")
	}

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
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
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft Id", "Error", err)
		return nil
	}
	searchInput := c.FormValue("search")
	log.Debug(c.Request().Context(), "Got request to search users")

	users, err := h.UserStore.SearchUsers(c.Request().Context(), searchInput, draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to search users", "Draft Id", draftId, "Search Input", searchInput, "Error", err)
		return nil
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to search for players in an invalid draft", "Draft Id", draftId, "Error", err)
		return nil
	}

	userUuid := c.Get("userUuid").(uuid.UUID)
	isOwner := userUuid == draftModel.Owner.UserUuid

	searchResults := draftView.PlayerSearchResults(users, draftId, isOwner, h.csrfToken(c))
	err = Render(c, searchResults)
	return err
}

func (h *Handler) InviteDraftPlayer(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	draftIdStr := c.Param("id")
	invitingUserUuid := userUuid
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid draft id", "Error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}
	invitedUserUuidString := c.FormValue("userUuid")
	invitedUserUuid, err := uuid.Parse(invitedUserUuidString)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse user guid", "Error", err)
		return c.String(http.StatusBadRequest, "Invalid user UUID")
	}

	// Check that the draft is in the correct state
	draft, err := h.DraftManager.GetDraft(draftId, false)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to load draft", "Draft Id", draftId, "Error", err)
		return err
	}

	if draft.GetStatus() != model.FILLING {
		return c.String(http.StatusBadRequest, "Draft must be in FILLING state to invite players")
	}

	isOwner := userUuid == draft.GetOwner().UserUuid
	if !isOwner {
		return c.String(http.StatusUnauthorized, "You must own the draft to invite a player")
	}

	_, err = h.DraftStore.InvitePlayer(c.Request().Context(), draftId, invitingUserUuid, invitedUserUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to invite player", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to invite player")
	}

	searchInput := c.FormValue("search")
	log.Debug(c.Request().Context(), "Got request to search users")

	users, err := h.UserStore.SearchUsers(c.Request().Context(), searchInput, draftId)

	if err != nil {
		log.Warn(c.Request().Context(), "Failed to search users", "Draft Id", draftId, "Search Input", searchInput, "Error", err)
		return nil
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to invite player to invalid draft", "Draft Id", draftId, "User Uuid", userUuid, "Error", err)
	}

	players := draftModel.Players

	updatedPage := draftView.UpdateAfterInvite(users, draftId, players, isOwner, h.csrfToken(c))
	err = Render(c, updatedPage)

	return err
}

func (h *Handler) HandleStartDraft(c echo.Context) error {
	draftIdStr := c.Param("id")
	log.Info(c.Request().Context(), "Got a request to start a draft", "Draft Id", draftIdStr)
	requestingUser := c.Get("userUuid").(uuid.UUID)
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Warn(c.Request().Context(), "Could not parse draftId", "Draft Id Str", draftIdStr, "Error", err)
		c.Response().Status = http.StatusBadRequest
		page := draftView.StartDraftButton(
			fmt.Sprintf("/u/draft/%d/startDraft", draftId),
			"Draft Id is not a number",
			false,
			h.csrfToken(c),
		)
		return Render(c, page)
	}

	draft, err := h.DraftManager.GetDraft(draftId, true)
	if err != nil {
		log.Warn(c.Request().Context(), "Could not load draft", "Draft Id", draftId, "Error", err)
		c.Response().Status = http.StatusBadRequest
		page := draftView.StartDraftButton(fmt.Sprintf("/u/draft/%d/startDraft", draftId), "Could not load draft", false, h.csrfToken(c))
		return Render(c, page)
	}

	if draft.GetOwner().UserUuid != requestingUser {
		log.Warn(c.Request().Context(), "User is not draft owner", "Draft Id", draftId, "User", requestingUser)
		c.Response().Status = http.StatusUnauthorized
		page := draftView.StartDraftButton(fmt.Sprintf("/u/draft/%d/startDraft", draftId), "Permission Denied", false, h.csrfToken(c))
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
		c.Response().Status = http.StatusBadRequest
		page := draftView.StartDraftButton(
			fmt.Sprintf("/u/draft/%d/startDraft", draftId),
			fmt.Sprintf("Draft must have exactly 8 accepted players to start (current: %d)", numAccepted),
			true,
			h.csrfToken(c),
		)
		return Render(c, page)
	}

	// Cancel the invites for players who have not accepted the draft
	if err := h.DraftStore.CancelOutstandingInvites(c.Request().Context(), draftId); err != nil {
		log.Error(c.Request().Context(), "Failed to cancel outstanding invites", "error", err, "draftId", draftId)
	}

	log.Info(c.Request().Context(), "Requesting draft state change to picking", "Draft Id", draftId)
	err = h.DraftManager.ExecuteDraftStateTransition(c.Request().Context(), draftId, model.WAITING_TO_START)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		return err
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
	return nil
}
