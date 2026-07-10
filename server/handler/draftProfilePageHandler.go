package handler

import (
	"fmt"
	"net/http"
	"server/draft"
	"server/log"
	"server/model"
	draftView "server/view/draft"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// HandleViewDraftProfile renders the draft profile page for the given draft ID.
func (h *Handler) HandleViewDraftProfile(c echo.Context) error {
	log.Debug(c.Request().Context(), "Got a request to serve the draft profile page")

	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id", "draftIdString", c.Param("id"), "error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}

	// TODO I think this should go through the draft manager
	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		//We want to redirect back to the home screen
		log.Debug(c.Request().Context(), "User attempted to visit incorrect draft id", "userUuid", userUuid, "draftId", draftId, "error", err)
		return c.Redirect(http.StatusSeeOther, "/u/home")
	}

	isOwner := userUuid == draftModel.Owner.UserUuid

	draftIndex := draftView.DraftProfileIndex(draftModel, isOwner, h.csrfToken(c))
	draftView := draftView.DraftProfile("Draft Profile", true, username, draftIndex, draftId, draftModel.DisplayName, isOwner)
	if err := Render(c, draftView); err != nil {
		log.Error(c.Request().Context(), "Handle View Draft Profile Failed To Render", "draftId", draftId, "error", err)
		return err
	}
	return nil
}

// HandleUpdateDraftProfile updates the draft profile fields (name, description, interval, times, webhook).
func (h *Handler) HandleUpdateDraftProfile(c echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to update a draft")

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Could not parse draftId from params", "error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}

	draftName := c.FormValue("draftName")
	description := c.FormValue("description")
	interval := c.FormValue("interval")
	discordWebhook := c.FormValue("discordWebhook")

	intInterval, err := strconv.Atoi(interval)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse interval", "error", err)
		return c.String(http.StatusBadRequest, "Invalid interval")
	}

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to write to invalid draft id", "userUuid", userUuid, "draftId", draftId, "error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}

	if draftModel.Owner.UserUuid != userUuid {
		//The user would need to hand craft this payload
		//so for now we just won't tell them what is wrong
		//because it is probably malicious
		log.Warn(c.Request().Context(), "User tried to update draft but was not the owner", "userUuid", userUuid, "draftId", draftId, "ownerId", draftModel.Owner.UserUuid)
		return c.String(http.StatusForbidden, "You do not have permission to update this draft")
	}

	draftModel = model.DraftModel{
		Id: draftId,
		Owner: model.User{
			UserUuid: userUuid,
		},
		DisplayName:    draftName,
		Description:    description,
		Interval:       intInterval,
		DiscordWebhook: discordWebhook,
	}

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get draft actor", "draftId", draftId, "error", err)
		return err
	}

	err = draft.UpdateDraft(c.Request().Context(), draftActor, draftModel)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to update draft", "draftId", draftId, "error", err)
		return err
	}

	log.Debug(c.Request().Context(), "Draft updated, reloading page", "draftId", draftId)
	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
	return nil
}

// SearchPlayers searches for users to invite to a draft and returns the results as HTML.
func (h *Handler) SearchPlayers(c echo.Context) error {
	currentUrl := c.Request().Header.Get("Hx-Current-Url")
	if currentUrl == "" {
		log.Warn(c.Request().Context(), "Missing Hx-Current-Url header")
		return c.String(http.StatusBadRequest, "Missing required header")
	}
	splitSource := strings.Split(currentUrl, "/")
	draftId, err := strconv.Atoi(splitSource[len(splitSource)-2])
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id", "error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}
	searchInput := c.FormValue("search")
	log.Debug(c.Request().Context(), "Got request to search users")

	users, err := h.UserStore.SearchUsers(c.Request().Context(), searchInput, draftId)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to search users", "draftId", draftId, "searchInput", searchInput, "error", err)
		return c.String(http.StatusInternalServerError, "Failed to search users")
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Debug(c.Request().Context(), "User attempted to search for players in an invalid draft", "draftId", draftId, "error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}

	userUuid := c.Get("userUuid").(uuid.UUID)
	isOwner := userUuid == draftModel.Owner.UserUuid

	searchResults := draftView.PlayerSearchResults(users, draftId, isOwner, h.csrfToken(c))
	if err := Render(c, searchResults); err != nil {
		log.Error(c.Request().Context(), "Search Players Failed To Render", "draftId", draftId, "error", err)
		return err
	}
	return nil
}

// InviteDraftPlayer invites a user to join the draft.
func (h *Handler) InviteDraftPlayer(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	draftIdStr := c.Param("id")
	invitingUserUuid := userUuid
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid draft id", "error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}
	invitedUserUuidString := c.FormValue("userUuid")
	invitedUserUuid, err := uuid.Parse(invitedUserUuidString)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse user guid", "error", err)
		return c.String(http.StatusBadRequest, "Invalid user UUID")
	}

	// Check that the draft is in the correct state
	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to load draft", "draftId", draftId, "error", err)
		return err
	}
	draftModel := draft.GetDraft(draftActor)

	if draftModel.Status != model.FILLING {
		log.Warn(c.Request().Context(), "User attempted to invite player to draft not in FILLING state", "draftId", draftId, "status", draftModel.Status)
		return c.String(http.StatusBadRequest, "Draft must be in FILLING state to invite players")
	}

	isOwner := userUuid == draftModel.Owner.UserUuid
	if !isOwner {
		log.Warn(c.Request().Context(), "User attempted to invite player to draft they do not own", "draftId", draftId, "userUuid", userUuid)
		return c.String(http.StatusUnauthorized, "You must own the draft to invite a player")
	}

	err = draft.InvitePlayer(c.Request().Context(), draftActor, model.DraftInvite{
		InvitingUserUuid: invitingUserUuid,
		InvitedUserUuid:  invitedUserUuid,
	})
	if err != nil {
		log.Error(c.Request().Context(), "Failed to invite player", "draftId", draftId, "invitedUserUuid", invitedUserUuid, "error", err)
		return c.String(http.StatusInternalServerError, "Failed to invite player")
	}
	log.Debug(c.Request().Context(), "Invited player to draft", "draftId", draftId, "invitedUserUuid", invitedUserUuid)

	searchInput := c.FormValue("search")

	users, err := h.UserStore.SearchUsers(c.Request().Context(), searchInput, draftId)

	if err != nil {
		log.Error(c.Request().Context(), "Failed to search users", "draftId", draftId, "searchInput", searchInput, "error", err)
		return c.String(http.StatusInternalServerError, "Failed to search users")
	}

	// Reload draft model from actor cache (updated by InvitePlayer)
	draftModel = draft.GetDraft(draftActor)
	players := draftModel.Players

	updatedPage := draftView.UpdateAfterInvite(users, draftId, players, isOwner, h.csrfToken(c))
	if err := Render(c, updatedPage); err != nil {
		log.Error(c.Request().Context(), "Invite Draft Player Failed To Render", "draftId", draftId, "error", err)
		return err
	}

	return nil
}

// HandleStartDraft transitions the draft from FILLING to WAITING_TO_START after validating 8 players.
func (h *Handler) HandleStartDraft(c echo.Context) error {
	draftIdStr := c.Param("id")
	log.Debug(c.Request().Context(), "Got a request to start a draft", "draftId", draftIdStr)
	requestingUser := c.Get("userUuid").(uuid.UUID)
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Warn(c.Request().Context(), "Could not parse draftId", "draftIdString", draftIdStr, "error", err)
		c.Response().Status = http.StatusBadRequest
		page := draftView.StartDraftButton(
			fmt.Sprintf("/u/draft/%d/startDraft", draftId),
			"Draft Id is not a number",
			false,
			h.csrfToken(c),
		)
		return Render(c, page)
	}

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Error(c.Request().Context(), "Could not load draft", "draftId", draftId, "error", err)
		c.Response().Status = http.StatusBadRequest
		page := draftView.StartDraftButton(fmt.Sprintf("/u/draft/%d/startDraft", draftId), "Could not load draft", false, h.csrfToken(c))
		return Render(c, page)
	}
	draftModel := draft.GetDraft(draftActor)

	if draftModel.Owner.UserUuid != requestingUser {
		log.Warn(c.Request().Context(), "User is not draft owner", "draftId", draftId, "userUuid", requestingUser)
		c.Response().Status = http.StatusUnauthorized
		page := draftView.StartDraftButton(fmt.Sprintf("/u/draft/%d/startDraft", draftId), "Permission Denied", false, h.csrfToken(c))
		return Render(c, page)
	}

	// Check that eight players have accepted the draft
	numAccepted := 0
	for _, p := range draftModel.Players {
		if !p.Pending {
			numAccepted++
		}
	}

	if numAccepted != 8 {
		log.Warn(c.Request().Context(), "User attempted to start a draft with an incorrect number of players", "draftId", draftId, "numAccepted", numAccepted)
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
		log.Error(c.Request().Context(), "Failed to cancel outstanding invites", "draftId", draftId, "error", err)
	}

	log.Debug(c.Request().Context(), "Requesting draft state change to picking", "draftId", draftId)
	err = draft.ExecuteDraftStateTransition(c.Request().Context(), draftActor, model.PICKING)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to execute draft state transition", "draftId", draftId, "error", err)
		return err
	}

	// Cancel the invites for players who have not accepted the draft
	if err := h.DraftStore.CancelOutstandingInvites(c.Request().Context(), draftId); err != nil {
		log.Error(c.Request().Context(), "Failed to cancel outstanding invites", "error", err, "draftId", draftId)
	}

	log.Info(c.Request().Context(), "Draft started", "draftId", draftId, "userUuid", requestingUser)
	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
	return nil
}
