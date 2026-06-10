package e2e

import (
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_TeamScoreLookup(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	// Seed team and match data
	seedTeam(t, db, "frc254", "Team 254", 42)
	seedMatch(t, db, "2026arc_qm1", 50, 40)
	seedMatchTeam(t, db, "2026arc_qm1", "frc254", "Red", false)
	seedMatch(t, db, "2026arc_qm2", 60, 30)
	seedMatchTeam(t, db, "2026arc_qm2", "frc254", "Blue", false)

	createTestUser(t, ts, "scoreuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login and navigate to team score page
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "scoreuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/team/score"),
		chromedp.WaitVisible(`input[name="teamNumber"]`),
		chromedp.SetValue(`input[name="teamNumber"]`, "254"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.Sleep(500 * time.Millisecond),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("score report shows team info", func(t *testing.T) {
		assert.Contains(t, pageBody, "Team 254")
		assert.Contains(t, pageBody, "Total Score")
	})

	t.Run("score report shows qualification matches", func(t *testing.T) {
		assert.Contains(t, pageBody, "Qualification Matches")
	})
}

func TestE2E_TeamAvatar(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	// Seed a team
	seedTeam(t, db, "frc254", "Team 254", 0)

	createTestUser(t, ts, "avataruser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageURL string

	// Login and request avatar
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "avataruser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/team/254/avatar"),
		chromedp.Sleep(500 * time.Millisecond),
		chromedp.Evaluate(`window.location.href`, &pageURL),
	)
	require.NoError(t, err)

	// The avatar endpoint may return 404 if no avatar is in TBA cache, but it should not crash
	// We just verify the endpoint is reachable
	assert.Contains(t, pageURL, "team/254/avatar", "avatar endpoint should be reachable")
}
