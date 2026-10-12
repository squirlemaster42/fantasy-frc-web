package handler

import (
	"errors"
	"net/http"
	"server/log"
	"server/model"
	"server/view/integrations"
	"uuid"

	"github.com/labstack/echo/v5"
)

func (h *Handler) HandleViewIntegrations(c *echo.Context) error {
	ctx := c.Request().Context()
	userUuid, err := h.requireUserUuid(c)
	if err != nil {
		return err
	}

	keys, err := h.Stores.ApiKeyStore.GetApiKeysForUser(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to get api keys for user", "UserUuid", userUuid, "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	index := integrations.IntegrationsIndex(keys, "", "", "", h.csrfToken(c))
	page := integrations.Integrations(" | Integrations", true, "", index)
	if err := Render(c, page); err != nil {
		log.Error(ctx, "HandleViewIntegrations failed to render", "Error", err)
	}
	return nil
}

func (h *Handler) HandleCreateIntegration(c *echo.Context) error {
	ctx := c.Request().Context()
	userUuid, err := h.requireUserUuid(c)
	if err != nil {
		return err
	}

	displayName := c.FormValue("displayName")
	if displayName == "" {
		keys, err := h.Stores.ApiKeyStore.GetApiKeysForUser(ctx, userUuid)
		if err != nil {
			log.Error(ctx, "Failed to get api keys for user", "UserUuid", userUuid, "Error", err)
			return c.String(http.StatusInternalServerError, "An error occurred")
		}
		index := integrations.IntegrationsIndex(keys, "Display name is required", "", "", h.csrfToken(c))
		if err := Render(c, index); err != nil {
			log.Error(ctx, "HandleCreateIntegration failed to render", "Error", err)
		}
		return nil
	}

	key, secret, err := h.Stores.ApiKeyStore.CreateApiKey(ctx, userUuid, displayName)
	if err != nil {
		log.Error(ctx, "Failed to create api key", "UserUuid", userUuid, "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	keys, err := h.Stores.ApiKeyStore.GetApiKeysForUser(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to get api keys for user", "UserUuid", userUuid, "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	index := integrations.IntegrationsIndex(keys, "", "Integration created. Copy your secret now — it will not be shown again.", secret+"|"+key.ClientId, h.csrfToken(c))
	if err := Render(c, index); err != nil {
		log.Error(ctx, "HandleCreateIntegration failed to render", "Error", err)
	}
	return nil
}

func (h *Handler) HandleRevokeIntegration(c *echo.Context) error {
	ctx := c.Request().Context()
	userUuid, err := h.requireUserUuid(c)
	if err != nil {
		return err
	}

	idParam := c.Param("id")
	keyId, err := uuid.Parse(idParam)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid integration id")
	}

	if err := h.Stores.ApiKeyStore.RevokeApiKey(ctx, keyId, userUuid); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return c.String(http.StatusNotFound, "Integration not found")
		}
		log.Error(ctx, "Failed to revoke api key", "KeyId", keyId, "UserUuid", userUuid, "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	return c.Redirect(http.StatusSeeOther, "/u/integrations")
}
