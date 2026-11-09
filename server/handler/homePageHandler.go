package handler

import (
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v5"

	"server/log"
	"server/model"
	"server/view"
)

func (h *Handler) HandleViewHome(c *echo.Context) error {
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
		return InternalServerError(c, "Failed to load drafts for user", err, "Search", search)
	}
	log.Debug(c.Request().Context(), "Loaded drafts for user", "username", username, "Search", search)

	homeIndex := view.HomeIndex(drafts, userUuid)
	home := view.Home("Draft Overview", true, username, homeIndex)
	return RenderOrLog(c, home, "Handle View Home Failed To Render", "Search", search)
}

func (h *Handler) HandleViewDraftList(c *echo.Context) error {
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
		return InternalServerError(c, "Failed to load drafts for user", err, "Page Number", page, "Search", search)
	}
	log.Debug(c.Request().Context(), "Loaded drafts for user", "username", username, "Page Number", page, "Search", search)

	var draftList templ.Component
	if page == 0 {
		if len(drafts) == 0 && search != "" {
			draftList = view.DraftSearchNoResults(search)
		} else {
			draftList = view.DraftListContainer(drafts, userUuid, 0, search)
		}
	} else {
		draftList = view.DraftList(drafts, userUuid, page, search)
	}
	return RenderOrLog(c, draftList, "Handle View Draft List Failed To Render", "Page Number", page, "Search", search)
}
