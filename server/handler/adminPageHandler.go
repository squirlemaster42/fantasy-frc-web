package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"server/log"
	"server/view/admin"
)

// adminService returns a new AdminService backed by the handler's dependencies.
func (h *Handler) adminService() *AdminService {
	return &AdminService{
		TBAHandler:     h.Services.TBAHandler,
		DraftStore:     h.Stores.DraftStore,
		TeamStore:      h.Stores.TeamStore,
		DraftActorMap:  h.Services.DraftActorMap,
		DraftDaemon:    h.Services.DraftDaemon,
		EventTeamCache: h.Services.EventTeamCache,
	}
}

// renderAdminResponse renders the result of an admin operation. It returns JSON
// when the request Accept header prefers application/json, otherwise it renders
// the AdminResult HTML partial.
func (h *Handler) renderAdminResponse(c *echo.Context, operation string, message string, err error) error {
	ctx := c.Request().Context()

	if err != nil {
		message = err.Error()
	}

	if strings.Contains(c.Request().Header.Get("Accept"), "application/json") {
		if jsonErr := c.JSON(http.StatusOK, map[string]any{
			"success": err == nil,
			"message": message,
		}); jsonErr != nil {
			return fmt.Errorf("failed to encode admin response: %w", jsonErr)
		}
		return nil
	}

	response := admin.AdminResult(operation, message, err != nil)
	if renderErr := Render(c, response); renderErr != nil {
		log.Error(ctx, "Failed to render admin result", "operation", operation, "error", renderErr)
		return renderErr
	}
	return nil
}

func (h *Handler) HandleAdminConsoleGet(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to render admin console")

	_, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	adminConsoleIndex := admin.AdminConsoleIndex(username, h.csrfToken(c))
	adminConsole := admin.AdminConsole(" | Admin Console", true, username, adminConsoleIndex)
	if err := Render(c, adminConsole); err != nil {
		log.Error(c.Request().Context(), "Failed to render admin console", "error", err)
		return err
	}
	return nil
}

func (h *Handler) HandleAdminPing(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin ping", "userUuid", userUuid)

	msg, svcErr := h.adminService().Ping(ctx)
	return h.renderAdminResponse(c, "Ping", msg, svcErr)
}

func (h *Handler) HandleAdminPopulateTeams(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin populate teams", "userUuid", userUuid)

	msg, svcErr := h.adminService().PopulateTeams(ctx)
	return h.renderAdminResponse(c, "Populate Teams", msg, svcErr)
}

func (h *Handler) HandleAdminListDrafts(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	search := c.FormValue("search")
	ctx := c.Request().Context()
	log.Info(ctx, "Admin list drafts", "userUuid", userUuid, "search", search)

	msg, svcErr := h.adminService().ListDrafts(ctx, search)
	return h.renderAdminResponse(c, "List Drafts", msg, svcErr)
}

func (h *Handler) HandleAdminStartDraft(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return h.renderAdminResponse(c, "Start Draft", "", fmt.Errorf("draft ID must be a number"))
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin start draft", "userUuid", userUuid, "draftId", draftId)

	msg, svcErr := h.adminService().StartDraft(ctx, draftId)
	return h.renderAdminResponse(c, "Start Draft", msg, svcErr)
}

func (h *Handler) HandleAdminSkipPickGlobal(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return h.renderAdminResponse(c, "Skip Pick", "", fmt.Errorf("draft ID must be a number"))
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin skip pick", "userUuid", userUuid, "draftId", draftId)

	msg, svcErr := h.adminService().SkipPick(ctx, draftId)
	return h.renderAdminResponse(c, "Skip Pick", msg, svcErr)
}

func (h *Handler) HandleAdminUndoPickGlobal(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return h.renderAdminResponse(c, "Undo Pick", "", fmt.Errorf("draft ID must be a number"))
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin undo pick", "userUuid", userUuid, "draftId", draftId)

	msg, svcErr := h.adminService().UndoPick(ctx, draftId)
	return h.renderAdminResponse(c, "Undo Pick", msg, svcErr)
}

func (h *Handler) HandleAdminModifyPickTime(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return h.renderAdminResponse(c, "Modify Pick Time", "", fmt.Errorf("draft ID must be a number"))
	}

	delta, err := time.ParseDuration(c.FormValue("time"))
	if err != nil {
		return h.renderAdminResponse(c, "Modify Pick Time", "", fmt.Errorf("invalid duration format. Use formats like 45m, 1h30m, 2h15m30s"))
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin modify pick time", "userUuid", userUuid, "draftId", draftId, "delta", delta)

	msg, svcErr := h.adminService().ModifyPickTime(ctx, draftId, delta)
	return h.renderAdminResponse(c, "Modify Pick Time", msg, svcErr)
}

func (h *Handler) HandleAdminAdminPick(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return h.renderAdminResponse(c, "Admin Pick", "", fmt.Errorf("draft ID must be a number"))
	}

	teamNumber := c.FormValue("team")
	if teamNumber == "" {
		return h.renderAdminResponse(c, "Admin Pick", "", fmt.Errorf("team number is required"))
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin make pick", "userUuid", userUuid, "draftId", draftId, "team", teamNumber)

	msg, svcErr := h.adminService().AdminPick(ctx, draftId, teamNumber)
	return h.renderAdminResponse(c, "Admin Pick", msg, svcErr)
}

func (h *Handler) HandleAdminRenameDraft(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return h.renderAdminResponse(c, "Rename Draft", "", fmt.Errorf("draft ID must be a number"))
	}

	newName := strings.TrimSpace(c.FormValue("name"))
	if newName == "" {
		return h.renderAdminResponse(c, "Rename Draft", "", fmt.Errorf("draft name cannot be empty"))
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin rename draft", "userUuid", userUuid, "draftId", draftId, "newName", newName)

	msg, svcErr := h.adminService().RenameDraft(ctx, draftId, newName)
	return h.renderAdminResponse(c, "Rename Draft", msg, svcErr)
}

func (h *Handler) HandleAdminViewWebhookKey(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin view webhook key", "userUuid", userUuid)

	msg, svcErr := h.adminService().ViewWebhookKey(ctx)
	return h.renderAdminResponse(c, "View Webhook Key", msg, svcErr)
}

func (h *Handler) HandleInvalidatePickCache(c *echo.Context) error {
	userUuid, _, err := h.requireUser(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	log.Info(ctx, "Admin invalidate pick cache", "userUuid", userUuid)

	msg, svcErr := h.adminService().InvalidatePickCache(ctx)
	return h.renderAdminResponse(c, "Invalidate Pick Cache", msg, svcErr)
}
