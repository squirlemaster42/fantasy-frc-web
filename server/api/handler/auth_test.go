package apihandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	apimodel "server/api/model"
	"server/model"
	"server/model/mocks"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	mocklib "github.com/stretchr/testify/mock"
)

func TestToken_Success(t *testing.T) {
	apiKeyStore := new(mocks.MockApiKeyStore)
	userUuid := uuid.New()
	signingKey := []byte("super-secret-key-that-is-32-bytes!")

	handler := NewAuthHandler(apiKeyStore, signingKey)

	reqBody, _ := json.Marshal(apimodel.TokenRequest{
		GrantType:    "client_credentials",
		ClientId:     "test_client_id",
		ClientSecret: "test_client_secret",
	})

	apiKeyStore.On("ValidateApiKey", mocklib.Anything, "test_client_id", "test_client_secret").Return(userUuid, nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Token(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp apimodel.TokenResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, int(15*time.Minute.Seconds()), resp.ExpiresIn)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestToken_InvalidGrantType(t *testing.T) {
	apiKeyStore := new(mocks.MockApiKeyStore)
	handler := NewAuthHandler(apiKeyStore, []byte("super-secret-key-that-is-32-bytes!"))

	reqBody, _ := json.Marshal(apimodel.TokenRequest{
		GrantType:    "password",
		ClientId:     "test_client_id",
		ClientSecret: "test_client_secret",
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Token(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToken_InvalidCredentials(t *testing.T) {
	apiKeyStore := new(mocks.MockApiKeyStore)
	handler := NewAuthHandler(apiKeyStore, []byte("super-secret-key-that-is-32-bytes!"))

	reqBody, _ := json.Marshal(apimodel.TokenRequest{
		GrantType:    "client_credentials",
		ClientId:     "test_client_id",
		ClientSecret: "wrong_secret",
	})

	apiKeyStore.On("ValidateApiKey", mocklib.Anything, "test_client_id", "wrong_secret").Return(uuid.UUID{}, model.ErrInvalidApiKey)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Token(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestToken_MissingCredentials(t *testing.T) {
	apiKeyStore := new(mocks.MockApiKeyStore)
	handler := NewAuthHandler(apiKeyStore, []byte("super-secret-key-that-is-32-bytes!"))

	reqBody, _ := json.Marshal(apimodel.TokenRequest{
		GrantType: "client_credentials",
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Token(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestToken_StoreError(t *testing.T) {
	apiKeyStore := new(mocks.MockApiKeyStore)
	handler := NewAuthHandler(apiKeyStore, []byte("super-secret-key-that-is-32-bytes!"))

	reqBody, _ := json.Marshal(apimodel.TokenRequest{
		GrantType:    "client_credentials",
		ClientId:     "test_client_id",
		ClientSecret: "test_client_secret",
	})

	apiKeyStore.On("ValidateApiKey", mocklib.Anything, "test_client_id", "test_client_secret").Return(uuid.UUID{}, errors.New("database error"))

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Token(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
