package apihandler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	apimodel "server/api/model"
	"server/model"
	"server/model/mocks"
	"testing"
	"time"
	"uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	mocklib "github.com/stretchr/testify/mock"
)

func authMiddleware(userUuid uuid.UUID) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("userUuid", userUuid)
			return next(c)
		}
	}
}

func setupIntegrationTest() (*Handler, *mocks.MockApiKeyStore, *echo.Echo) {
	apiKeyStore := new(mocks.MockApiKeyStore)
	handler := NewHandler(nil, nil, apiKeyStore, nil, nil)
	e := echo.New()
	return handler, apiKeyStore, e
}

func newIntegrationContext(e *echo.Echo, userUuid uuid.UUID, method, path string, body []byte) *echo.Context {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userUuid", userUuid)
	return c
}

func responseRecorder(c *echo.Context) *httptest.ResponseRecorder {
	return c.Response().(*echo.Response).Unwrap().(*httptest.ResponseRecorder)
}

func TestListIntegrations_Success(t *testing.T) {
	handler, apiKeyStore, e := setupIntegrationTest()
	userUuid := uuid.New()
	keys := []model.ApiKey{
		{
			Id:          uuid.New(),
			UserUuid:    userUuid,
			DisplayName: "Test Key",
			ClientId:    "client_1",
			CreatedAt:   time.Now().UTC(),
		},
	}

	apiKeyStore.On("GetApiKeysForUser", mocklib.Anything, userUuid).Return(keys, nil)

	c := newIntegrationContext(e, userUuid, http.MethodGet, "/api/v1/integrations", nil)
	err := handler.ListIntegrations(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, responseRecorder(c).Code)

	var resp []apimodel.IntegrationResponse
	assert.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, "Test Key", resp[0].DisplayName)
	assert.Equal(t, "client_1", resp[0].ClientId)
}

func TestCreateIntegration_Success(t *testing.T) {
	handler, apiKeyStore, e := setupIntegrationTest()
	userUuid := uuid.New()
	key := &model.ApiKey{
		Id:          uuid.New(),
		UserUuid:    userUuid,
		DisplayName: "New Integration",
		ClientId:    "ffrc_new_client",
		CreatedAt:   time.Now().UTC(),
	}

	apiKeyStore.On("CreateApiKey", mocklib.Anything, userUuid, "New Integration").Return(key, "secret", nil)

	reqBody, _ := json.Marshal(apimodel.CreateIntegrationRequest{DisplayName: "New Integration"})
	c := newIntegrationContext(e, userUuid, http.MethodPost, "/api/v1/integrations", reqBody)
	err := handler.CreateIntegration(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, responseRecorder(c).Code)

	var resp apimodel.CreateIntegrationResponse
	assert.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	assert.Equal(t, "New Integration", resp.Integration.DisplayName)
	assert.Equal(t, "ffrc_new_client", resp.ClientId)
	assert.Equal(t, "secret", resp.ClientSecret)
}

func TestCreateIntegration_MissingDisplayName(t *testing.T) {
	handler, _, e := setupIntegrationTest()
	userUuid := uuid.New()

	reqBody, _ := json.Marshal(apimodel.CreateIntegrationRequest{})
	c := newIntegrationContext(e, userUuid, http.MethodPost, "/api/v1/integrations", reqBody)
	err := handler.CreateIntegration(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, responseRecorder(c).Code)
}

func TestRevokeIntegration_Success(t *testing.T) {
	handler, apiKeyStore, e := setupIntegrationTest()
	userUuid := uuid.New()
	keyId := uuid.New()

	apiKeyStore.On("RevokeApiKey", mocklib.Anything, keyId, userUuid).Return(nil)

	e.POST("/api/v1/integrations/:id/revoke", handler.RevokeIntegration, authMiddleware(userUuid))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+keyId.String()+"/revoke", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRevokeIntegration_InvalidId(t *testing.T) {
	handler, _, e := setupIntegrationTest()
	userUuid := uuid.New()

	e.POST("/api/v1/integrations/:id/revoke", handler.RevokeIntegration, authMiddleware(userUuid))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/invalid/revoke", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
