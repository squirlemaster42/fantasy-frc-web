package e2e

import (
	"testing"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_UserProfilePage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "profileuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login and navigate to user profile
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "profileuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/userProfile"),
		chromedp.WaitVisible(`#profile-box`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("user profile shows username", func(t *testing.T) {
		assert.Contains(t, pageBody, "profileuser")
		assert.Contains(t, pageBody, "User Profile")
	})

	t.Run("discord id field present", func(t *testing.T) {
		assert.Contains(t, pageBody, `name="discordId"`)
	})

	t.Run("password change fields present", func(t *testing.T) {
		assert.Contains(t, pageBody, `name="currentPassword"`)
		assert.Contains(t, pageBody, `name="newPassword"`)
		assert.Contains(t, pageBody, `name="confirmNewPassword"`)
	})

	t.Run("save changes button present", func(t *testing.T) {
		assert.Contains(t, pageBody, "Save Changes")
		assert.Contains(t, pageBody, `hx-post="/u/userProfile"`)
	})
}

func TestE2E_UserProfileUpdate(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "updateuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login and update discord ID
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "updateuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/userProfile"),
		chromedp.WaitVisible(`input[name="discordId"]`),
		chromedp.SendKeys(`input[name="discordId"]`, "12345678901234567"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`[role="alert"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("profile updated successfully", func(t *testing.T) {
		assert.Contains(t, pageBody, "Profile updated successfully")
	})
}
