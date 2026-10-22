package handler

import (
	"fmt"
	"net/http"
	"server/log"
	"server/model"
	"server/types"
	draftView "server/view/draft"

	"github.com/labstack/echo/v5"
)

func (h *Handler) HandleViewCreateDraft(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to serve the create draft page")

	_, username, err := h.requireUser(c)
	if err != nil {
		return err
	}
	draftModel := model.DraftModel{
		Id:     -1,
		Status: model.FILLING,
	}

	draftCreateIndex := draftView.DraftProfileIndex(draftModel, true, h.csrfToken(c))
	draftCreate := draftView.DraftProfile(" | Create Draft", true, username, draftCreateIndex, types.NewPageData(-1, "", true))
	if err := Render(c, draftCreate); err != nil {
		log.Error(c.Request().Context(), "Handle View Draft Create Failed To Render", "error", err)
		return err
	}
	return nil
}

func (h *Handler) HandleCreateDraftPost(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to create a draft")
	draftName := c.FormValue("draftName")
	description := c.FormValue("description")

	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	timingType, initialPickTimeSec, incrementTimeSec, perPickExpTimeSec, err := parseDraftTimingForm(c, model.DraftModel{
		TimingType:         model.PER_PICK,
		InitialPickTimeSec: 3600,
		IncrementTimeSec:   900,
		PerPickExpTimeSec:  10800,
	})
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid draft timing values", "userUuid", userUuid, "error", err)
		return c.String(http.StatusBadRequest, err.Error())
	}

	draftModel := model.DraftModel{
		Owner: model.User{
			UserUuid: userUuid,
		},
		DisplayName:        draftName,
		Description:        description,
		Status:             model.FILLING,
		TimingType:         timingType,
		InitialPickTimeSec: initialPickTimeSec,
		IncrementTimeSec:   incrementTimeSec,
		PerPickExpTimeSec:  perPickExpTimeSec,
	}

	draftId, err := h.Stores.DraftStore.CreateDraft(c.Request().Context(), &draftModel)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to create draft", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create draft")
	}
	log.Info(c.Request().Context(), "Draft created", "draftId", draftId, "username", username)
	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/u/draft/%d/profile", draftId))
	return nil
}
