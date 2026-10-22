package apihandler

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	apimodel "server/api/model"
	apimiddleware "server/api/middleware"
	"server/database"
	"server/draft"
	"server/model"
	"server/utils"
	"sync"
	"testing"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"uuid"
)

var (
	integrationDB     *sql.DB
	integrationDBOnce sync.Once
	integrationDBErr  error
)

func setupIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	integrationDBOnce.Do(func() {
		_ = godotenv.Load(filepath.Join("..", "..", ".env"))

		dbUsername := os.Getenv("DB_USERNAME")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbIp := os.Getenv("DB_IP")
		dbName := os.Getenv("DB_NAME")

		if dbUsername == "" || dbPassword == "" || dbIp == "" || dbName == "" {
			integrationDBErr = sql.ErrConnDone
			return
		}

		integrationDB, integrationDBErr = database.RegisterDatabaseConnection(context.Background(), dbUsername, dbPassword, dbIp, dbName)
	})

	if errors.Is(integrationDBErr, sql.ErrConnDone) {
		t.Skip("Skipping test: database credentials not found in environment")
	}
	require.NoError(t, integrationDBErr)
	require.NotNil(t, integrationDB)

	return integrationDB
}

func createIntegrationUser(t *testing.T, db *sql.DB) model.User {
	t.Helper()

	store := model.NewSQLUserStore(db)
	ctx := context.Background()

	username := "api_test_" + randomString(8)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	require.NoError(t, err)

	userUuid, err := store.RegisterUser(ctx, username, string(passwordHash))
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Users WHERE UserUuid = $1", userUuid)
	})

	return model.User{UserUuid: userUuid, Username: username}
}

func createIntegrationDraft(t *testing.T, db *sql.DB, owner model.User) model.DraftModel {
	t.Helper()

	store := model.NewSQLDraftStore(db)
	ctx := context.Background()

	draft := &model.DraftModel{
		DisplayName:       "API Test Draft " + randomString(8),
		Description:       "Integration test draft",
		Owner:             owner,
		Status:            model.FILLING,
		TimingType:        model.INCREMENT,
		IncrementTimeSec:  60,
		PerPickExpTimeSec: 300,
	}

	draftId, err := store.CreateDraft(ctx, draft)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Drafts WHERE Id = $1", draftId)
	})

	draft.Id = draftId
	return *draft
}

func addIntegrationPlayer(t *testing.T, db *sql.DB, draftId int, player model.User) {
	t.Helper()

	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO DraftPlayers (draftId, UserUuid) VALUES ($1, $2)", draftId, player.UserUuid)
	require.NoError(t, err)
}

func setupIntegrationAPI(t *testing.T) (*echo.Echo, *sql.DB, []byte) {
	t.Helper()

	db := setupIntegrationDB(t)
	signingKey := []byte("test-signing-key-that-is-32-bytes!")

	pickConfig := utils.DefaultPickWindowConfig()
	actorMap := draft.NewDraftActorMap(model.NewSQLDraftStore(db), nil, nil, nil, nil, pickConfig, 128)

	handler := NewHandler(model.NewSQLDraftStore(db), model.NewSQLUserStore(db), model.NewSQLApiKeyStore(db), model.NewSQLTeamStore(db), actorMap)
	authHandler := NewAuthHandler(model.NewSQLApiKeyStore(db), signingKey)

	e := echo.New()
	apiV1 := e.Group("/api/v1")

	apiV1.POST("/auth/token", authHandler.Token)

	protected := apiV1.Group("")
	protected.Use(apimiddleware.JWTAuth(signingKey))

	protected.GET("/drafts", handler.ListDrafts, apimiddleware.RequireScopes(apimiddleware.ScopeDraftsRead))
	protected.POST("/drafts", handler.CreateDraft, apimiddleware.RequireScopes(apimiddleware.ScopeDraftsWrite))
	protected.GET("/drafts/:id", handler.GetDraft, apimiddleware.RequireScopes(apimiddleware.ScopeDraftsRead))
	protected.PATCH("/drafts/:id", handler.UpdateDraft, apimiddleware.RequireScopes(apimiddleware.ScopeDraftsWrite))
	protected.POST("/drafts/:id/start", handler.StartDraft, apimiddleware.RequireScopes(apimiddleware.ScopeDraftsWrite))
	protected.POST("/drafts/:id/invites", handler.CreateInvite, apimiddleware.RequireScopes(apimiddleware.ScopeInvitesWrite))
	protected.POST("/invites/:id/accept", handler.AcceptInvite, apimiddleware.RequireScopes(apimiddleware.ScopeInvitesWrite))
	protected.POST("/drafts/:id/admin/skip-pick", handler.AdminSkipPick, apimiddleware.RequireScopes(apimiddleware.ScopeAdmin))
	protected.POST("/drafts/:id/admin/undo-pick", handler.AdminUndoPick, apimiddleware.RequireScopes(apimiddleware.ScopeAdmin))

	return e, db, signingKey
}

func createIntegrationAPIKey(t *testing.T, db *sql.DB, userUuid uuid.UUID, scopes []string) (clientId, clientSecret string) {
	t.Helper()

	store := model.NewSQLApiKeyStore(db)
	key, secret, err := store.CreateApiKey(context.Background(), userUuid, "Integration Test Key", scopes)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM UserApiKeys WHERE Id = $1", key.Id)
	})

	return key.ClientId, secret
}

func getIntegrationToken(t *testing.T, e *echo.Echo, clientId, clientSecret string) string {
	t.Helper()

	//nolint:gosec
	reqBody, err := json.Marshal(apimodel.TokenRequest{
		GrantType:    "client_credentials",
		ClientId:     clientId,
		ClientSecret: clientSecret,
	})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/token", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp apimodel.TokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)

	return resp.AccessToken
}

func TestCreateInvite_Integration(t *testing.T) {
	e, db, _ := setupIntegrationAPI(t)
	owner := createIntegrationUser(t, db)
	player := createIntegrationUser(t, db)
	d := createIntegrationDraft(t, db, owner)

	clientId, secret := createIntegrationAPIKey(t, db, owner.UserUuid, []string{"invites:write", "drafts:read"})
	token := getIntegrationToken(t, e, clientId, secret)

	reqBody, err := json.Marshal(apimodel.CreateInviteRequest{UserUuid: player.UserUuid.String()})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/drafts/"+itoa(d.Id)+"/invites", bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestStartDraft_Integration(t *testing.T) {
	e, db, _ := setupIntegrationAPI(t)
	owner := createIntegrationUser(t, db)
	d := createIntegrationDraft(t, db, owner)

	for range 7 {
		player := createIntegrationUser(t, db)
		addIntegrationPlayer(t, db, d.Id, player)
	}

	clientId, secret := createIntegrationAPIKey(t, db, owner.UserUuid, []string{"drafts:write"})
	token := getIntegrationToken(t, e, clientId, secret)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/drafts/"+itoa(d.Id)+"/start", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp apimodel.DraftResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Picking", string(resp.Status))
}

func itoa(n int) string {
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if i == len(buf) {
		return "0"
	}
	return string(buf[i:])
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			b[i] = letters[i%len(letters)]
			continue
		}
		b[i] = letters[idx.Int64()]
	}
	return string(b)
}
