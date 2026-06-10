package e2e

import (
	"testing"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_ServerStarts(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	// Verify the server responds
	var body string
	ctx, cancel := newBrowserContext(t)
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/healthz"),
		chromedp.Text("body", &body),
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", body)
}

func TestE2E_LoginFlow(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	// Seed a test user
	createTestUser(t, ts, "testuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.WaitVisible(`input[name="password"]`),
		chromedp.SendKeys(`input[name="username"]`, "testuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	// Should be on the home page
	assert.Contains(t, pageBody, "Create New Draft")
	assert.Contains(t, pageBody, "testuser")
}

func TestE2E_InvalidLogin(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "testuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "testuser"),
		chromedp.SendKeys(`input[name="password"]`, "wrongpassword"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`[role="alert"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	// Should stay on login page and show error
	assert.Contains(t, pageBody, "You have entered an invalid username or password")
	assert.Contains(t, pageBody, `name="username"`)
	assert.Contains(t, pageBody, `name="password"`)
}

func TestE2E_LandingPageNavigation(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var url string
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/"),
		chromedp.WaitVisible(`a[href="/login"]`),
		chromedp.Click(`a[href="/login"]`),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.Location(&url),
	)
	require.NoError(t, err)
	assert.Contains(t, url, "/login")
}

func TestE2E_RegisterPage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/register"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.WaitVisible(`input[name="password"]`),
		chromedp.WaitVisible(`input[name="confirmPassword"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	assert.Contains(t, pageBody, `hx-post="/register"`)
	assert.Contains(t, pageBody, `name="username"`)
	assert.Contains(t, pageBody, `name="password"`)
	assert.Contains(t, pageBody, `name="confirmPassword"`)
	assert.Contains(t, pageBody, `Create Account`)
}

func TestE2E_RegisterAndLogin(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Register a new user
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/register"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "newuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.SendKeys(`input[name="confirmPassword"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)
	assert.Contains(t, pageBody, "Create New Draft")
	assert.Contains(t, pageBody, "newuser")
}
