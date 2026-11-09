package handler

import (
	"strconv"

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
	return RenderOrLog(c, draftCreate, "Handle View Draft Create Failed To Render")
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
		return BadRequest(c, "Invalid draft timing values", err.Error(), "userUuid", userUuid, "error", err)
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
		return InternalServerError(c, "Failed to create draft", err)
	}
	log.Info(c.Request().Context(), "Draft created", "draftId", draftId, "username", username)
	return HXRedirect(c, "/u/draft/"+strconv.Itoa(draftId)+"/profile")
}
