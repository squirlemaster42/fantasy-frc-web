package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"server/model"
)

func TestE2E_DraftGameplayLoop(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx := context.Background()

	// Create owner — additional players are created by createDraftWithPlayers
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "gameloopowner", "Password123")

	// Create draft with 8 players and transition to PICKING
	draftId := createPickingDraftWithPlayers(t, ts, ownerUuid, "Gameplay Draft", 8)

	// Seed TBA cache for 64 unique teams so all picks pass validation
	validEvents := []string{"2026arc", "2026cur", "2026dal", "2026gal", "2026hop", "2026joh", "2026mil", "2026new", "2026cmptx"}
	for i := 0; i < 64; i++ {
		teamNum := 100 + i
		seedTbaCache(t, db, fmt.Sprintf("frc%d", teamNum), validEvents)
	}

	// Seed match data for all 64 teams so scores show up later
	for i := 0; i < 64; i++ {
		teamNum := 100 + i
		tbaId := fmt.Sprintf("frc%d", teamNum)
		seedTeam(t, db, tbaId, fmt.Sprintf("Team %d", teamNum), 10)
		seedMatch(t, db, fmt.Sprintf("2026arc_qm%d", i), 50, 40)
		seedMatchTeam(t, db, fmt.Sprintf("2026arc_qm%d", i), tbaId, "Red", false)
	}

	// Make first 4 picks via UI to prove the user flow
	t.Run("ui picks work", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			// Get current pick from DB
			draftModel, err := ts.Handler.DraftStore.GetDraft(ctx, draftId)
			require.NoError(t, err)
			currentPickerUuid := draftModel.NextPick.User.UserUuid

			// Look up the username for this UUID
			pickerUsername, err := ts.Handler.UserStore.GetUsername(ctx, currentPickerUuid)
			require.NoError(t, err)
			require.NotEmpty(t, pickerUsername, "could not find picker username")

			// Make pick via UI
			teamNum := 100 + i
			makePickViaUI(t, ts, pickerUsername, draftId, teamNum)
		}
	})

	// Make remaining 60 picks via HTTP to keep the test fast
	t.Run("bulk picks complete draft", func(t *testing.T) {
		// Pre-login all users so we can reuse sessions across picks
		clients := map[string]*http.Client{
			"gameloopowner": loginViaHTTP(t, ts, "gameloopowner"),
		}
		for i := 1; i < 8; i++ {
			username := fmt.Sprintf("player%d", i)
			clients[username] = loginViaHTTP(t, ts, username)
		}

		for i := 4; i < 64; i++ {
			// Get current pick from DB
			draftModel, err := ts.Handler.DraftStore.GetDraft(ctx, draftId)
			require.NoError(t, err)
			currentPickerUuid := draftModel.NextPick.User.UserUuid

			// Look up the username for this UUID
			pickerUsername, err := ts.Handler.UserStore.GetUsername(ctx, currentPickerUuid)
			require.NoError(t, err)
			require.NotEmpty(t, pickerUsername, "could not find picker username")

			client, ok := clients[pickerUsername]
			require.True(t, ok, "no pre-logged client for user %s", pickerUsername)

			teamNum := 100 + i
			makePickViaHTTP(t, ts, client, draftId, teamNum)
		}
		picks, err := ts.Handler.DraftStore.GetPicks(ctx, draftId)
		require.NoError(t, err)
		require.Len(t, picks, 64, "expected 64 picks after bulk picks")
	})

	// Verify draft transitioned to TEAMS_PLAYING
	t.Run("draft transitions to teams playing", func(t *testing.T) {
		var draftModel model.DraftModel
		var err error
		// Poll until draft transitions (draft actor processes picks asynchronously)
		for i := 0; i < 30; i++ {
			draftModel, err = ts.Handler.DraftStore.GetDraft(ctx, draftId)
			require.NoError(t, err)
			if draftModel.Status == model.TEAMS_PLAYING {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		assert.Equal(t, model.TEAMS_PLAYING, draftModel.Status)
	})

	// Navigate to draft score page and verify scores
	t.Run("score page shows actual scores", func(t *testing.T) {
		ctx2, cancel := newBrowserContext(t)
		defer cancel()

		var pageBody string

		// Login as owner
		err := chromedp.Run(ctx2,
			chromedp.Navigate(ts.BaseURL+"/login"),
			chromedp.WaitVisible(`input[name="username"]`),
			chromedp.SendKeys(`input[name="username"]`, "gameloopowner"),
			chromedp.SendKeys(`input[name="password"]`, "Password123"),
			chromedp.Click(`button[type="submit"]`),
			chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/draftScore", draftId)),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.OuterHTML("body", &pageBody),
		)
		require.NoError(t, err)

		// Should show actual scores, not the "not yet available" message
		assert.Contains(t, pageBody, "Draft Scores")
		assert.NotContains(t, pageBody, "Scores will be available once teams start playing")
		assert.Contains(t, pageBody, "Total Score")
	})
}

// makePickViaUI logs in as the given user and submits a pick via the browser.
func makePickViaUI(t *testing.T, ts *TestServer, username string, draftId, teamNum int) {
	t.Helper()

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

		err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, username),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/pick", draftId)),
		chromedp.WaitVisible(`input[name="pickInput"]`),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Click(`input[name="pickInput"]`),
		chromedp.SendKeys(`input[name="pickInput"]`, fmt.Sprintf("%d", teamNum)),
		chromedp.WaitReady(`button[type="submit"]`),
		chromedp.Click(`button[type="submit"]`),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	// Should not show error
	assert.NotContains(t, pageBody, "Could Not Make Pick")
	assert.NotContains(t, pageBody, `id="pickError"`)
	assert.NotContains(t, pageBody, "you must be the picking player")
	assert.Contains(t, pageBody, fmt.Sprintf("%d", teamNum), "picked team should appear in the pick table")
}

// extractCsrfToken extracts the CSRF token value from HTML containing name="csrf_token".
func extractCsrfToken(bodyStr string) string {
	if idx := strings.Index(bodyStr, `name="csrf_token"`); idx != -1 {
		valIdx := strings.Index(bodyStr[idx:], `value="`)
		if valIdx != -1 {
			start := idx + valIdx + 7
			end := strings.Index(bodyStr[start:], `"`)
			if end != -1 {
				return bodyStr[start : start+end]
			}
		}
	}
	return ""
}

// loginViaHTTP logs in via HTTP and returns a client with the session cookie.
func loginViaHTTP(t *testing.T, ts *TestServer, username string) *http.Client {
	t.Helper()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	resp, err := client.Get(ts.BaseURL + "/login")
	require.NoError(t, err)
	loginBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)

	csrfToken := extractCsrfToken(string(loginBody))
	require.NotEmpty(t, csrfToken, "failed to extract CSRF token from login page")

	loginData := url.Values{}
	loginData.Set("username", username)
	loginData.Set("password", "Password123")
	loginData.Set("csrf_token", csrfToken)
	resp, err = client.PostForm(ts.BaseURL+"/login", loginData)
	require.NoError(t, err)
	resp.Body.Close()

	require.NotEmpty(t, resp.Header.Get("HX-Redirect"), "login failed for user %s", username)
	return client
}

// makePickViaHTTP makes a pick using a pre-authenticated HTTP client.
func makePickViaHTTP(t *testing.T, ts *TestServer, client *http.Client, draftId, teamNum int) {
	t.Helper()

	// Get pick page to extract CSRF token
	resp, err := client.Get(ts.BaseURL + fmt.Sprintf("/u/draft/%d/pick", draftId))
	require.NoError(t, err)
	pickBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)

	csrfToken := extractCsrfToken(string(pickBody))
	require.NotEmpty(t, csrfToken, "failed to extract CSRF token from pick page")

	// Submit pick
	pickData := url.Values{}
	pickData.Set("pickInput", fmt.Sprintf("%d", teamNum))
	pickData.Set("csrf_token", csrfToken)
	resp, err = client.PostForm(ts.BaseURL+fmt.Sprintf("/u/draft/%d/makePick", draftId), pickData)
	require.NoError(t, err)
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "pick request failed")
	bodyStr := string(bodyBytes)
	require.NotContains(t, bodyStr, "Could Not Make Pick", "pick failed for team %d", teamNum)
	require.NotContains(t, bodyStr, `id="pickError"`, "pick failed for team %d", teamNum)
	require.NotContains(t, bodyStr, "you must be the picking player to make a pick", "wrong picker for team %d", teamNum)
	require.NotContains(t, bodyStr, "attempting to make pick that is not the current pick", "stale current pick for team %d", teamNum)
}
