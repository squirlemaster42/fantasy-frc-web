package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_TeamScorePage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "teamuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login and navigate to team score page
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "teamuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/team/score"),
		chromedp.WaitVisible(`input[name="teamNumber"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("team score page shows title", func(t *testing.T) {
		assert.Contains(t, pageBody, "Team Score Lookup")
	})

	t.Run("team score search form present", func(t *testing.T) {
		assert.Contains(t, pageBody, `name="teamNumber"`)
		assert.Contains(t, pageBody, `type="number"`)
		assert.Contains(t, pageBody, "Get Scores")
	})

	t.Run("csrf token present", func(t *testing.T) {
		assert.Contains(t, pageBody, `name="csrf_token"`)
	})
}

func TestE2E_DraftScorePage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx := context.Background()
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "scoreowner", "Password123")

	draftId := createTestDraft(t, ts, ownerUuid, "Score Draft", "For score testing")

	ctx2, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login and navigate to draft score page
	err := chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "scoreowner"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/draftScore", draftId)),
		chromedp.Sleep(1000),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("draft score page shows title", func(t *testing.T) {
		assert.Contains(t, pageBody, "Draft Scores")
	})

	t.Run("score not yet available message", func(t *testing.T) {
		// Draft is in FILLING state, so scores are not available yet
		assert.Contains(t, pageBody, "Scores will be available once teams start playing")
	})
}

func TestE2E_AdminConsolePage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestAdmin(t, ts, "adminuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login as admin and navigate to admin console
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "adminuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/admin/console"),
		chromedp.WaitVisible(`#commandInput`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("admin console shows terminal", func(t *testing.T) {
		assert.Contains(t, pageBody, `id="commandInput"`)
		assert.Contains(t, pageBody, `id="output"`)
		assert.Contains(t, pageBody, "adminuser@webapp~$")
	})

	t.Run("admin command input present", func(t *testing.T) {
		assert.Contains(t, pageBody, `name="command"`)
		assert.Contains(t, pageBody, `hx-post="/u/admin/processCommand"`)
	})
}

func TestE2E_AdminConsolePingCommand(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestAdmin(t, ts, "admincmd", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string
	var ok bool

	// Login as admin, navigate to console, and run ping command
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "admincmd"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/admin/console"),
		chromedp.WaitVisible(`#commandInput`),
		chromedp.SendKeys(`#commandInput`, "ping"),
		chromedp.SendKeys(`#commandInput`, "\n"), // Press Enter to trigger HTMX
		chromedp.Poll(`document.querySelector('#output').innerText.includes('Pong')`, &ok, chromedp.WithPollingInterval(100*time.Millisecond), chromedp.WithPollingTimeout(5*time.Second)),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("ping command returns pong", func(t *testing.T) {
		assert.Contains(t, pageBody, "ping")
		assert.Contains(t, pageBody, "Pong")
	})
}

func TestE2E_AdminConsoleNonAdminRedirect(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "regularuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login as non-admin and try to access admin console
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "regularuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/admin/console"),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("non-admin redirected to home", func(t *testing.T) {
		assert.Contains(t, pageBody, "regularuser")
		assert.Contains(t, pageBody, "Create New Draft")
	})
}
