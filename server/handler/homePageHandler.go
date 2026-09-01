package handler

import (
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"

	"server/log"
	"server/model"
	"server/view"
)

func (h *Handler) HandleViewHome(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	search := c.QueryParam("search")

	log.Debug(c.Request().Context(), "Loading drafts for user", "username", username, "Search", search)
	drafts, err := h.Stores.DraftStore.SearchDrafts(c.Request().Context(), model.DraftSearchQuery{
		UserUuid:        userUuid,
		DraftNameSearch: search,
		PageSize:        20,
		PageNum:         0,
	})
	if err != nil {
		log.Error(c.Request().Context(), "Failed to load drafts for user", "error", err, "Search", search)
		return c.String(http.StatusInternalServerError, "Failed to load drafts")
	}
	log.Debug(c.Request().Context(), "Loaded drafts for user", "username", username, "Search", search)

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

	search := c.QueryParam("search")

	page := 0
	if pageStr := c.QueryParam("page"); pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Page number must be a positive number")
		}
	}

	log.Debug(c.Request().Context(), "Loading drafts for user", "username", username, "Page Number", page, "Search", search)
	drafts, err := h.Stores.DraftStore.SearchDrafts(c.Request().Context(), model.DraftSearchQuery{
		UserUuid:        userUuid,
		DraftNameSearch: search,
		PageSize:        20,
		PageNum:         page,
	})
	if err != nil {
		log.Error(c.Request().Context(), "Failed to load drafts for user", "error", err, "Page Number", page, "Search", search)
		return c.String(http.StatusInternalServerError, "Failed to load drafts")
	}
	log.Debug(c.Request().Context(), "Loaded drafts for user", "username", username, "Page Number", page, "Search", search)

	var draftList templ.Component
	if page == 0 {
		draftList = view.DraftSearchResults(drafts, userUuid, 0, search)
	} else {
		draftList = view.DraftList(drafts, userUuid, page, search)
	}
	if err := Render(c, draftList); err != nil {
		log.Error(c.Request().Context(), "Handle View Draft List Failed To Render", "error", err, "Page Number", page, "Search", search)
		return err
	}

	return nil
}
