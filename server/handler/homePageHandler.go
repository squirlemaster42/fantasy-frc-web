package handler

import (
	"net/http"
	"server/log"
	"server/model"
	"server/view"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewHome(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	log.Debug(c.Request().Context(), "Loading drafts for user", "username", username)
	drafts, err := h.Stores.DraftStore.SearchDrafts(c.Request().Context(), model.DraftSearchQuery{
		UserUuid: userUuid,
		PageSize: 20,
		PageNum: 0,
	})
	if err != nil {
		log.Error(c.Request().Context(), "Failed to load drafts for user", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load drafts")
	}
	log.Debug(c.Request().Context(), "Loaded drafts for user", "username", username)

	homeIndex := view.HomeIndex(drafts, userUuid)
	home := view.Home("Draft Overview", true, username, homeIndex)
	if err := Render(c, home); err != nil {
		log.Error(c.Request().Context(), "Handle View Home Failed To Render", "error", err)
		return err
	}
	return nil
}

func (h *Handler) HandleViewDraftList(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

    page, err := echo.QueryParam[int](c, "page")
    if err != nil || page < 0 {
        return echo.NewHTTPError(http.StatusBadRequest, "Page number must be a positive number")
    }

	log.Debug(c.Request().Context(), "Loading drafts for user", "username", username, "Page Number", page)
	drafts, err := h.Stores.DraftStore.SearchDrafts(c.Request().Context(), model.DraftSearchQuery{
		UserUuid: userUuid,
		PageSize: 20,
		PageNum: page,
	})
	if err != nil {
		log.Error(c.Request().Context(), "Failed to load drafts for user", "error", err, "Page Number", page)
		return c.String(http.StatusInternalServerError, "Failed to load drafts")
	}
	log.Debug(c.Request().Context(), "Loaded drafts for user", "username", username, "Page Number", page)

	draftList := view.DraftList(drafts, userUuid, page)
	if err := Render(c, draftList); err != nil {
		log.Error(c.Request().Context(), "Handle View Draft List Failed To Render", "error", err, "Page Number", page)
		return err
	}

	return nil
}
