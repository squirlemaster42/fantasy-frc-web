package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"server/model"
)

// createTestDraft creates a draft directly in the database for E2E testing.
func createTestDraft(t *testing.T, ts *TestServer, ownerUuid uuid.UUID, name, description string) int {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	draft := &model.DraftModel{
		DisplayName:    name,
		Description:    description,
		Interval:       60,
		StartTime:      now.Add(1 * time.Hour),
		EndTime:        now.Add(72 * time.Hour),
		Status:         model.FILLING,
		Owner:          model.User{UserUuid: ownerUuid},
		DiscordWebhook: "",
	}
	id, err := ts.Handler.DraftStore.CreateDraft(ctx, draft)
	require.NoError(t, err)
	return id
}

// createPickingDraft creates a draft and transitions it to PICKING state for E2E testing.
func createPickingDraft(t *testing.T, ts *TestServer, ownerUuid uuid.UUID, name string) int {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	draft := &model.DraftModel{
		DisplayName:    name,
		Description:    "Picking draft",
		Interval:       60,
		StartTime:      now.Add(-1 * time.Hour),
		EndTime:        now.Add(72 * time.Hour),
		Status:         model.FILLING,
		Owner:          model.User{UserUuid: ownerUuid},
		DiscordWebhook: "",
	}
	draftId, err := ts.Handler.DraftStore.CreateDraft(ctx, draft)
	require.NoError(t, err)

	// Transition to WAITING_TO_START
	err = ts.Handler.DraftStore.UpdateDraftStatus(ctx, draftId, model.WAITING_TO_START)
	require.NoError(t, err)

	// Randomize pick order (owner is already a non-pending player)
	err = ts.Handler.DraftStore.RandomizePickOrder(ctx, draftId)
	require.NoError(t, err)

	// Make first pick available
	nextPlayer, err := ts.Handler.DraftStore.NextPick(ctx, draftId)
	require.NoError(t, err)
	_, err = ts.Handler.DraftStore.MakePickAvailable(ctx, nextPlayer.Id, time.Now(), now.Add(1*time.Hour))
	require.NoError(t, err)

	// Transition to PICKING
	err = ts.Handler.DraftStore.UpdateDraftStatus(ctx, draftId, model.PICKING)
	require.NoError(t, err)

	return draftId
}

func TestE2E_ViewDraftProfile(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "draftowner", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "draftowner"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
	)
	require.NoError(t, err)

	// Navigate to create draft
	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/u/createDraft"),
		chromedp.WaitVisible(`input[name="draftName"]`),
	)
	require.NoError(t, err)

	// Create a draft via the UI
	// The handler expects format: 2006-01-02T15:04:05 (with seconds)
	now := time.Now()
	startTime := now.Add(1 * time.Hour).Format("2006-01-02T15:04:05")
	endTime := now.Add(72 * time.Hour).Format("2006-01-02T15:04:05")

	err = chromedp.Run(ctx,
		chromedp.SetValue(`input[name="draftName"]`, "Test Draft View"),
		chromedp.SetValue(`textarea[name="description"]`, "A draft for viewing"),
		chromedp.SetValue(`input[name="startTime"]`, startTime),
		chromedp.SetValue(`input[name="endTime"]`, endTime),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`input[name="search"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("draft profile shows draft name", func(t *testing.T) {
		assert.Contains(t, pageBody, "Test Draft View")
	})

	t.Run("draft profile shows description", func(t *testing.T) {
		assert.Contains(t, pageBody, "A draft for viewing")
	})

	t.Run("owner sees save button", func(t *testing.T) {
		assert.Contains(t, pageBody, "Save Changes")
	})

	t.Run("owner sees invite section", func(t *testing.T) {
		assert.Contains(t, pageBody, "Invite Players")
		assert.Contains(t, pageBody, `name="search"`)
	})

	t.Run("owner sees start draft button", func(t *testing.T) {
		assert.Contains(t, pageBody, "Start Draft")
	})

	t.Run("draft settings visible", func(t *testing.T) {
		assert.Contains(t, pageBody, "Draft Settings")
		assert.Contains(t, pageBody, `name="interval"`)
		assert.Contains(t, pageBody, `name="startTime"`)
		assert.Contains(t, pageBody, `name="endTime"`)
	})
}

func TestE2E_DraftProfileNavigation(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	// Register two users
	ctx := context.Background()
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "owner2", "Password123")
	_, _ = ts.Handler.UserStore.RegisterUser(ctx, "otheruser", "Password123")

	// Create a draft for owner2
	draftId := createTestDraft(t, ts, ownerUuid, "Nav Draft", "For nav testing")

	ctx2, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login as owner2
	err := chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "owner2"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
	)
	require.NoError(t, err)

	// Navigate directly to draft profile
	err = chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/profile", draftId)),
		chromedp.WaitVisible(`input[name="draftName"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("draft profile loads", func(t *testing.T) {
		assert.Contains(t, pageBody, "Nav Draft")
		assert.Contains(t, pageBody, "For nav testing")
	})
}

func TestE2E_UpdateDraftSettings(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx := context.Background()
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "updateowner", "Password123")
	draftId := createTestDraft(t, ts, ownerUuid, "Update Draft", "For update testing")

	ctx2, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login and navigate to draft profile
	err := chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "updateowner"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/profile", draftId)),
		chromedp.WaitVisible(`input[name="draftName"]`),
	)
	require.NoError(t, err)

	// Update draft name and interval
	err = chromedp.Run(ctx2,
		chromedp.SetValue(`input[name="draftName"]`, "Updated Draft Name"),
		chromedp.SetValue(`input[name="interval"]`, "120"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/profile", draftId)),
		chromedp.WaitVisible(`input[name="draftName"]`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("draft name updated", func(t *testing.T) {
		assert.Contains(t, pageBody, "Updated Draft Name")
	})

	t.Run("draft settings visible after update", func(t *testing.T) {
		assert.Contains(t, pageBody, "120")
	})
}

func TestE2E_SearchPlayers(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx := context.Background()
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "searchowner", "Password123")
	_, _ = ts.Handler.UserStore.RegisterUser(ctx, "searchtarget", "Password123")
	draftId := createTestDraft(t, ts, ownerUuid, "Search Draft", "For search testing")

	ctx2, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login and navigate to draft profile
	err := chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "searchowner"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/profile", draftId)),
		chromedp.WaitVisible(`input[name="search"]`),
	)
	require.NoError(t, err)

	// Search for the other user
	err = chromedp.Run(ctx2,
		chromedp.SetValue(`input[name="search"]`, "searchtarget"),
		chromedp.SendKeys(`input[name="search"]`, "\n"),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("search results show invited user", func(t *testing.T) {
		assert.Contains(t, pageBody, "searchtarget")
	})
}
