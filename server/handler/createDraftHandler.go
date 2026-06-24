package handler

import (
	"fmt"
	"net/http"
	"server/log"
	"server/model"
	"server/view/draft"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewCreateDraft(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to serve the create draft page")

	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}
	draftModel := model.DraftModel{
		Id: -1,
	}

	draftCreateIndex := draft.DraftProfileIndex(draftModel, true, h.csrfToken(c))
	draftCreate := draft.DraftProfile(" | Create Draft", true, username, draftCreateIndex, -1, true)
	err = Render(c, draftCreate)
	if err != nil {
		log.Error(c.Request().Context(), "Handle View Draft Create Failed To Render", "Error", err)
	}
	return nil
}

func (h *Handler) HandleCreateDraftPost(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to create a draft")
	draftName := c.FormValue("draftName")
	description := c.FormValue("description")
	interval := c.FormValue("interval")

	userUuid := c.Get("userUuid").(uuid.UUID)

	intInterval, err := strconv.Atoi(interval)

	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Interval must be a number, was %s", interval))
	}

	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	draftModel := model.DraftModel{
		Owner: model.User{
			UserUuid: userUuid,
		},
		DisplayName: draftName,
		Description: description,
		Interval:    intInterval,
		Status:      model.FILLING,
	}

	log.Info(c.Request().Context(), "Created Draft for user", "User", username)

	draftId, err := h.DraftStore.CreateDraft(c.Request().Context(), &draftModel)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to create draft", "Interval", intInterval, "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create draft")
	}
	log.Info(c.Request().Context(), "Draft created. Redirecting to /u/draft/:draftId:/profile", "Draft Id", draftId)
	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
	return nil
}
