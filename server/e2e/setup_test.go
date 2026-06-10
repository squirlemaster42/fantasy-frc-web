package e2e

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"server/assets"
	"server/authentication"
	"server/cache"
	"server/database"
	"server/discord"
	"server/draft"
	"server/handler"
	"server/middleware"
	"server/model"
	"server/picking"
	"server/tbaHandler"
)

// mockTbaTransport returns 304 Not Modified for all TBA requests so seeded cache is used.
type mockTbaTransport struct{}

func (m *mockTbaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusNotModified,
		Header:     http.Header{},
		Body:       http.NoBody,
		Request:    req,
	}, nil
}

// mockDiscordStore implements model.DiscordStore for E2E tests.
type mockDiscordStore struct{}

func (m *mockDiscordStore) GetPlayerDiscordId(ctx context.Context, draftPlayerId int) (sql.NullString, error) {
	return sql.NullString{}, nil
}

func (m *mockDiscordStore) GetDraftWebhook(ctx context.Context, draftId int) (string, error) {
	return "", nil
}

// TestServer holds the echo server and test database for e2e tests.
type TestServer struct {
	Echo       *echo.Echo
	DB         *sql.DB
	BaseURL    string
	Handler    *handler.Handler
	DiscordBus *discord.DiscordWebhookBus
	Shutdown   func()
}

// setupTestDatabase starts a Docker PostgreSQL container, runs migrations, and returns a connection.
// Cleanup must be called when done.
func setupTestDatabase(t *testing.T) (*sql.DB, string, func()) {
	t.Helper()

	// Check prerequisites
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping e2e test")
	}
	if _, err := exec.LookPath("goose"); err != nil {
		t.Skip("goose CLI not found in PATH, skipping e2e test")
	}

	ctx := context.Background()

	// Find an available port to avoid conflicts
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "failed to find available port")
	dbPort := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	listener.Close()

	containerName := fmt.Sprintf("fantasy-frc-e2e-%d", time.Now().UnixNano())
	dbName := "fantasyfrc_e2e"
	dbUser := "fantasyfrc"
	dbPass := "testpassword"

	// Start PostgreSQL container
	cmd := exec.CommandContext(ctx,
		"docker", "run", "-d",
		"--name", containerName,
		"-e", "POSTGRES_DB="+dbName,
		"-e", "POSTGRES_USER="+dbUser,
		"-e", "POSTGRES_PASSWORD="+dbPass,
		"-p", dbPort+":5432",
		"postgres:16",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to start postgres container: %s", out)

	// Wait for PostgreSQL to be ready
	ready := false
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		checkCmd := exec.CommandContext(ctx,
			"docker", "exec", containerName,
			"pg_isready", "-U", dbUser,
		)
		if err := checkCmd.Run(); err == nil {
			ready = true
			break
		}
	}
	require.True(t, ready, "postgres container did not become ready")

	// Additional delay to ensure postgres is fully accepting connections
	time.Sleep(2 * time.Second)

	// Run migrations with retry — pg_isready can report true before connections are accepted
	migrationsDir := "../../database/migrations"
	connStr := fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s?sslmode=disable", dbUser, dbPass, dbPort, dbName)
	var gooseOut []byte
	for i := 0; i < 5; i++ {
		gooseCmd := exec.CommandContext(ctx,
			"goose", "-dir", migrationsDir, "postgres", connStr, "up",
		)
		gooseOut, err = gooseCmd.CombinedOutput()
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.NoError(t, err, "failed to run migrations: %s", gooseOut)

	// Connect to database
	db, err := database.RegisterDatabaseConnection(ctx, dbUser, dbPass, "localhost:"+dbPort, dbName)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		exec.Command("docker", "stop", containerName).Run()
		exec.Command("docker", "rm", "-f", containerName).Run()
	}

	return db, connStr, cleanup
}

// createTestServer sets up an Echo server with the test database.
func createTestServer(t *testing.T, db *sql.DB) *TestServer {
	t.Helper()

	userStore := model.NewSQLUserStore(db)
	draftStore := model.NewSQLDraftStore(db)
	teamStore := model.NewSQLTeamStore(db)

	minPasswordLength := 8
	csrfSecret := "test-csrf-secret-for-e2e-tests-only-32-bytes"
	secureHttpCookie := false

	// Create minimal test server
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Static assets
	e.Add(http.MethodGet, "/css/*", echo.StaticDirectoryHandler(assets.CSS(), false))
	e.Add(http.MethodGet, "/img/*", echo.StaticDirectoryHandler(assets.Img(), false))
	e.Add(http.MethodGet, "/js/*", echo.StaticDirectoryHandler(assets.JS(), false))

	// Pick notifier for draft actor map
	pickNotifier := &picking.PickNotifier{
		Watchers: make(map[int][]picking.Watcher),
	}

	// TBA handler for pick validation
	tbaH := tbaHandler.NewHandler("dummy-token", db)
	// Use a mock HTTP client that always returns 304 so seeded TbaCache is used
	tbaH.SetClient(&http.Client{
		Transport: &mockTbaTransport{},
	})

	discordBus := discord.NewBus()

	draftActorMap := draft.NewDraftActorMap(draftStore, tbaH, &mockDiscordStore{}, discordBus, pickNotifier)

	// Avatar store (no Redis in e2e tests, falls back to TBA)
	avatarStore, err := cache.NewAvatarStore(*tbaH, "", "", 0)
	require.NoError(t, err)

	// Create handler
	h := &handler.Handler{
		UserStore:         userStore,
		DraftStore:        draftStore,
		TeamStore:         teamStore,
		TbaHandler:        *tbaH,
		DraftActorMap:     draftActorMap,
		SecureHttpCookie:  secureHttpCookie,
		MinPasswordLength: minPasswordLength,
		CsrfSecret:        csrfSecret,
		AvatarStore:       &avatarStore,
	}

	// Setup auth
	auth := authentication.NewAuth(userStore)
	csrf := middleware.NewCSRF(csrfSecret, secureHttpCookie)

	// Public routes
	e.GET("/", h.HandleViewLanding)
	e.GET("/login", h.HandleViewLogin)
	e.POST("/login", h.HandleLoginPost)
	e.GET("/register", h.HandleViewRegister)
	e.POST("/register", h.HandlerRegisterPost)
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Protected routes (minimal middleware matching production)
	protected := e.Group("/u", auth.Authenticate, csrf.CSRF())
	protected.POST("/logout", h.HandleLogoutPost)
	protected.GET("/home", h.HandleViewHome)
	protected.GET("/createDraft", h.HandleViewCreateDraft)
	protected.POST("/createDraft", h.HandleCreateDraftPost)
	protected.GET("/draft/:id/profile", h.HandleViewDraftProfile)
	protected.POST("/draft/:id/updateDraft", h.HandleUpdateDraftProfile)
	protected.POST("/draft/:id/startDraft", h.HandleStartDraft)
	protected.GET("/draft/:id/pick", h.ServePickPage)
	protected.POST("/draft/:id/makePick", h.HandlerPickRequest)
	protected.GET("/draft/:id/pickNotifier", h.PickNotifier)
	protected.POST("/draft/:id/invitePlayer", h.InviteDraftPlayer)
	protected.GET("/viewInvites", h.HandleViewInvites)
	protected.POST("/acceptInvite", h.HandleAcceptInvite)
	protected.POST("/draft/:id/skipPickToggle", h.HandleSkipPickToggle)
	protected.GET("/userProfile", h.HandleViewUserProfile)
	protected.POST("/userProfile", h.HandleUpdateUserProfile)
	protected.GET("/team/score", h.HandleTeamScore)
	protected.POST("/team/score", h.HandleGetTeamScore)
	protected.GET("/draft/:id/draftScore", h.HandleDraftScore)
	protected.GET("/draft/:id/team/:teamNumber", h.HandleDraftTeamScore)
	protected.GET("/draft/:id/admin", h.HandleDraftAdminGet)
	protected.POST("/draft/:id/admin/skipPick", h.HandleAdminSkipPick)
	protected.POST("/draft/:id/admin/extendTime", h.HandleAdminExtendTime)
	protected.POST("/draft/:id/admin/makePick", h.HandleAdminMakePick)
	protected.POST("/draft/:id/admin/undoPick", h.HandleAdminUndoPick)
	protected.POST("/searchPlayers", h.SearchPlayers)
	protected.GET("/team/:id/avatar", h.GetTeamAvatar)
	admin := protected.Group("/admin", auth.CheckAdmin)
	admin.GET("/console", h.HandleAdminConsoleGet)
	admin.POST("/processCommand", h.HandleRunCommand)

	// Start server on a random port
	srv := httptest.NewServer(e)

	shutdown := func() {
		srv.Close()
		discordBus.Stop()
	}

	return &TestServer{
		Echo:       e,
		DB:         db,
		BaseURL:    srv.URL,
		Handler:    h,
		DiscordBus: discordBus,
		Shutdown:   shutdown,
	}
}

// createTestUser creates a user in the test database and returns their credentials.
func createTestUser(t *testing.T, ts *TestServer, username, password string) {
	t.Helper()
	ctx := context.Background()
	_, err := ts.Handler.UserStore.RegisterUser(ctx, username, password)
	require.NoError(t, err)
}

// createTestAdmin creates a user and sets IsAdmin = true in the database.
func createTestAdmin(t *testing.T, ts *TestServer, username, password string) {
	t.Helper()
	ctx := context.Background()
	uuid, err := ts.Handler.UserStore.RegisterUser(ctx, username, password)
	require.NoError(t, err)

	_, err = ts.DB.ExecContext(ctx, "UPDATE Users SET IsAdmin = true WHERE UserUuid = $1", uuid)
	require.NoError(t, err)
}

// loginAsUser performs a browser login and returns the chromedp context with the session.
func loginAsUser(t *testing.T, ctx context.Context, ts *TestServer, username, password string) context.Context {
	t.Helper()

	var pageBody string
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, username),
		chromedp.SendKeys(`input[name="password"]`, password),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)
	require.Contains(t, pageBody, "Create New Draft", "login should redirect to home page")

	return ctx
}

// newBrowserContext creates a new headless Chrome context.
// Set CHROME_HEADED=1 to run headed for debugging.
func newBrowserContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	// chromedp.DefaultExecAllocatorOptions includes Headless by default
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	if os.Getenv("CHROME_HEADED") == "1" {
		// Remove headless flag for debugging
		var noHeadless []chromedp.ExecAllocatorOption
		for _, opt := range opts {
			// chromedp.Headless sets --headless; we skip it when headed
			if fmt.Sprintf("%p", opt) != fmt.Sprintf("%p", chromedp.Headless) {
				noHeadless = append(noHeadless, opt)
			}
		}
		opts = noHeadless
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)

	return ctx, func() {
		cancel()
		allocCancel()
	}
}

// skipIfNoDocker skips the test if Docker is unavailable.
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found, skipping e2e test")
	}
	// Quick check that docker daemon is responsive
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping e2e test")
	}
}

// skipIfNoChrome skips the test if Chrome is unavailable.
func skipIfNoChrome(t *testing.T) {
	t.Helper()
	// chromedp will auto-download a browser if none is installed, but we
	// prefer the system Chrome. chromedp will handle the download.
	// This helper just skips if CHROME_SKIP is set.
	if os.Getenv("CHROME_SKIP") == "1" {
		t.Skip("CHROME_SKIP=1 set, skipping e2e test")
	}
}

// containsIgnoreCase checks if the string contains the substring (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// seedTbaCache inserts a cached TBA response so MakeEventListReq returns without network calls.
func seedTbaCache(t *testing.T, db *sql.DB, teamTbaId string, events []string) {
	t.Helper()
	ctx := context.Background()
	url := "https://www.thebluealliance.com/api/v3/team/" + teamTbaId + "/events/2026/keys"
	body, _ := json.Marshal(events)
	_, err := db.ExecContext(ctx, "INSERT INTO TbaCache (url, etag, responseBody) VALUES ($1, $2, $3) ON CONFLICT (url) DO UPDATE SET etag = EXCLUDED.etag, responseBody = EXCLUDED.responseBody", url, "etag", body)
	require.NoError(t, err)
}

// seedTeam inserts a team into the Teams table.
func seedTeam(t *testing.T, db *sql.DB, tbaId, name string, allianceScore int) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO Teams (tbaId, name, allianceScore) VALUES ($1, $2, $3) ON CONFLICT (tbaId) DO UPDATE SET name = EXCLUDED.name, allianceScore = EXCLUDED.allianceScore", tbaId, name, allianceScore)
	require.NoError(t, err)
}

// seedMatch inserts a match into the Matches table.
func seedMatch(t *testing.T, db *sql.DB, matchTbaId string, redscore, bluescore int) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO Matches (tbaId, redscore, bluescore) VALUES ($1, $2, $3) ON CONFLICT (tbaId) DO UPDATE SET redscore = EXCLUDED.redscore, bluescore = EXCLUDED.bluescore", matchTbaId, redscore, bluescore)
	require.NoError(t, err)
}

// seedMatchTeam inserts a match-team association into the Matches_Teams table.
func seedMatchTeam(t *testing.T, db *sql.DB, matchTbaId, teamTbaId, alliance string, isDqed bool) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO Matches_Teams (match_tbaId, team_tbaId, alliance, isdqed) VALUES ($1, $2, $3, $4) ON CONFLICT (match_tbaId, team_tbaId) DO UPDATE SET alliance = EXCLUDED.alliance, isdqed = EXCLUDED.isdqed", matchTbaId, teamTbaId, alliance, isDqed)
	require.NoError(t, err)
}

// createDraftWithPlayers creates a draft and adds the specified number of players (including owner).
func createDraftWithPlayers(t *testing.T, ts *TestServer, ownerUuid uuid.UUID, name string, numPlayers int) int {
	t.Helper()
	ctx := context.Background()

	draftId := createTestDraft(t, ts, ownerUuid, name, " gameplay test draft")

	// Invite and accept additional players
	for i := 1; i < numPlayers; i++ {
		username := fmt.Sprintf("player%d", i)
		userUuid, err := ts.Handler.UserStore.RegisterUser(ctx, username, "Password123")
		require.NoError(t, err)

		inviteId, err := ts.Handler.DraftStore.InvitePlayer(ctx, draftId, ownerUuid, userUuid)
		require.NoError(t, err)

		_, _, err = ts.Handler.DraftStore.AcceptInvite(ctx, inviteId)
		require.NoError(t, err)

		err = ts.Handler.DraftStore.AddPlayerToDraft(ctx, draftId, userUuid)
		require.NoError(t, err)
	}

	return draftId
}

// startDraft transitions a draft from FILLING to PICKING via the draft actor.
func startDraft(t *testing.T, ts *TestServer, draftId int) {
	t.Helper()
	ctx := context.Background()

	draftActor, err := ts.Handler.DraftActorMap.GetActor(ctx, draftId)
	require.NoError(t, err)

	err = draft.ExecuteDraftStateTransition(ctx, draftActor, model.WAITING_TO_START)
	require.NoError(t, err)

	err = draft.ExecuteDraftStateTransition(ctx, draftActor, model.PICKING)
	require.NoError(t, err)
}

// createPickingDraftWithPlayers creates a draft with multiple players and transitions it to PICKING directly.
func createPickingDraftWithPlayers(t *testing.T, ts *TestServer, ownerUuid uuid.UUID, name string, numPlayers int) int {
	t.Helper()
	ctx := context.Background()

	draftId := createDraftWithPlayers(t, ts, ownerUuid, name, numPlayers)

	err := ts.Handler.DraftStore.UpdateDraftStatus(ctx, draftId, model.WAITING_TO_START)
	require.NoError(t, err)

	err = ts.Handler.DraftStore.RandomizePickOrder(ctx, draftId)
	require.NoError(t, err)

	nextPlayer, err := ts.Handler.DraftStore.NextPick(ctx, draftId)
	require.NoError(t, err)

	_, err = ts.Handler.DraftStore.MakePickAvailable(ctx, nextPlayer.Id, time.Now(), time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	err = ts.Handler.DraftStore.UpdateDraftStatus(ctx, draftId, model.PICKING)
	require.NoError(t, err)

	return draftId
}
