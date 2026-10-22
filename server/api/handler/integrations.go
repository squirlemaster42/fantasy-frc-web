package apihandler

import (
	"fmt"
	"net/http"
	"server/api"
	apimodel "server/api/model"
	"server/log"
	"server/model"
	"uuid"

	"github.com/labstack/echo/v5"
)

// ListIntegrations returns all API keys owned by the authenticated user.
func (h *Handler) ListIntegrations(c *echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	keys, err := h.ApiKeyStore.GetApiKeysForUser(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to list integrations", "UserUuid", userUuid, "Error", err)
		api.InternalError(c.Response())
		return err
	}

	response := make([]apimodel.IntegrationResponse, 0, len(keys))
	for _, key := range keys {
		response = append(response, mapApiKey(key))
	}

	return respondJSON(c, http.StatusOK, response)
}

// CreateIntegration creates a new API key for the authenticated user.
func (h *Handler) CreateIntegration(c *echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	var req apimodel.CreateIntegrationRequest
	if err := c.Bind(&req); err != nil {
		api.BadRequest(c.Response(), "Invalid request body")
		return fmt.Errorf("failed to bind request body: %w", err)
	}
	if req.DisplayName == "" {
		api.BadRequest(c.Response(), "display_name is required")
		return nil
	}

	key, secret, err := h.ApiKeyStore.CreateApiKey(ctx, userUuid, req.DisplayName, req.Scopes)
	if err != nil {
		log.Error(ctx, "Failed to create integration", "UserUuid", userUuid, "Error", err)
		api.InternalError(c.Response())
		return err
	}

	return respondJSON(c, http.StatusCreated, apimodel.CreateIntegrationResponse{
		Integration:  mapApiKey(*key),
		ClientId:     key.ClientId,
		ClientSecret: secret,
	})
}

// RevokeIntegration revokes an API key owned by the authenticated user.
func (h *Handler) RevokeIntegration(c *echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	idParam := c.Param("id")
	keyId, err := uuid.Parse(idParam)
	if err != nil {
		api.BadRequest(c.Response(), "Invalid integration id")
		return fmt.Errorf("failed to parse integration id: %w", err)
	}

	if err := h.ApiKeyStore.RevokeApiKey(ctx, keyId, userUuid); err != nil {
		log.Error(ctx, "Failed to revoke integration", "KeyId", keyId, "UserUuid", userUuid, "Error", err)
		api.InternalError(c.Response())
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func mapApiKey(key model.ApiKey) apimodel.IntegrationResponse {
	r := apimodel.IntegrationResponse{
		Id:          key.Id.String(),
		DisplayName: key.DisplayName,
		ClientId:    key.ClientId,
		Scopes:      key.Scopes,
		CreatedAt:   key.CreatedAt,
	}
	if key.LastUsedAt.Valid {
		r.LastUsedAt = key.LastUsedAt.Time
	}
	return r
}
