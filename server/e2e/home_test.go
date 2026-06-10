package e2e

import (
	"testing"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_HomePage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "homeuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login and land on home page
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "homeuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("home page shows username", func(t *testing.T) {
		assert.Contains(t, pageBody, "homeuser")
	})

	t.Run("create draft button present", func(t *testing.T) {
		assert.Contains(t, pageBody, "Create New Draft")
		assert.Contains(t, pageBody, `href="/u/createDraft"`)
	})

	t.Run("empty state when no drafts", func(t *testing.T) {
		assert.Contains(t, pageBody, "No Drafts Yet")
	})
}

func TestE2E_CreateDraftPage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "draftuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "draftuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
	)
	require.NoError(t, err)

	// Navigate to create draft page
	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/u/createDraft"),
		chromedp.WaitVisible(`input[name="draftName"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("create draft form present", func(t *testing.T) {
		assert.Contains(t, pageBody, `name="draftName"`)
		assert.Contains(t, pageBody, `name="description"`)
		assert.Contains(t, pageBody, `name="interval"`)
		assert.Contains(t, pageBody, `name="startTime"`)
		assert.Contains(t, pageBody, `name="endTime"`)
		assert.Contains(t, pageBody, `type="submit"`)
		assert.Contains(t, pageBody, "Save Changes")
		assert.Contains(t, pageBody, "Invite Players")
	})
}

func TestE2E_AuthRedirect(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "redirectuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Try to access a protected page without logging in
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/u/home"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("redirected to login when not authenticated", func(t *testing.T) {
		assert.Contains(t, pageBody, "Sign In")
		assert.Contains(t, pageBody, `name="username"`)
	})

	// Now login and verify we can access the home page
	err = chromedp.Run(ctx,
		chromedp.SendKeys(`input[name="username"]`, "redirectuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("home page accessible after login", func(t *testing.T) {
		assert.Contains(t, pageBody, "redirectuser")
		assert.Contains(t, pageBody, "Create New Draft")
	})
}
