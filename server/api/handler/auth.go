package apihandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"server/api"
	apimodel "server/api/model"
	"server/authentication/jwt"
	"server/log"
	"server/model"
	"time"

	"github.com/labstack/echo/v5"
)

const (
	accessTokenDuration = 15 * time.Minute
)

type AuthHandler struct {
	ApiKeyStore   model.ApiKeyStore
	JwtSigningKey []byte
}

func NewAuthHandler(apiKeyStore model.ApiKeyStore, jwtSigningKey []byte) *AuthHandler {
	return &AuthHandler{
		ApiKeyStore:   apiKeyStore,
		JwtSigningKey: jwtSigningKey,
	}
}

func (h *AuthHandler) Token(c *echo.Context) error {
	ctx := c.Request().Context()

	var req apimodel.TokenRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		api.InvalidGrant(c.Response(), "Invalid request body")
		return fmt.Errorf("failed to decode token request: %w", err)
	}

	if req.GrantType != "client_credentials" {
		api.InvalidGrant(c.Response(), "Unsupported grant_type")
		return nil
	}
	if req.ClientId == "" || req.ClientSecret == "" {
		api.InvalidClient(c.Response(), "client_id and client_secret are required")
		return nil
	}

	userUuid, scopes, err := h.ApiKeyStore.ValidateApiKey(ctx, req.ClientId, req.ClientSecret)
	if err != nil {
		if errors.Is(err, model.ErrInvalidApiKey) {
			api.InvalidClient(c.Response(), "Invalid client credentials")
			return nil
		}
		log.Error(ctx, "Failed to validate api key", "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	token, err := jwt.Sign(userUuid, h.JwtSigningKey, accessTokenDuration, scopes)
	if err != nil {
		log.Error(ctx, "Failed to sign jwt", "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	if err := c.JSON(http.StatusOK, apimodel.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(accessTokenDuration.Seconds()),
	}); err != nil {
		return fmt.Errorf("failed to write token response: %w", err)
	}
	return nil
}
