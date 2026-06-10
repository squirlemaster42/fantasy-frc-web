package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_DraftAdminPage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(context.Background(), "adminowner", "Password123")
	draftId := createPickingDraftWithPlayers(t, ts, ownerUuid, "Admin Draft", 2)

	// Seed TBA cache for a team and add team to database
	seedTbaCache(t, db, "frc254", []string{"2026arc", "2026cur", "2026dal", "2026gal", "2026hop", "2026joh", "2026mil", "2026new", "2026cmptx"})
	seedTeam(t, db, "frc254", "The Cheesy Poofs", 100)

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login as owner and navigate to admin page
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "adminowner"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/admin", draftId)),
		chromedp.WaitVisible(`#adminMessage`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("admin page shows draft status", func(t *testing.T) {
		assert.Contains(t, pageBody, "Draft Administration")
		assert.Contains(t, pageBody, "Picking")
	})

	t.Run("admin page shows current pick", func(t *testing.T) {
		assert.Contains(t, pageBody, "Current Pick")
	})

	// Test skip pick
	t.Run("admin skip pick works", func(t *testing.T) {
		err := chromedp.Run(ctx,
			chromedp.Click(`button[hx-post*="/admin/skipPick"]`),
			chromedp.Poll("document.querySelector('#adminMessage').textContent.includes('Pick skipped successfully')", nil, chromedp.WithPollingInterval(100*time.Millisecond)),
			chromedp.OuterHTML("body", &pageBody),
		)
		require.NoError(t, err)
		assert.Contains(t, pageBody, "Pick skipped successfully")
	})

	// Test admin make pick
	t.Run("admin make pick works", func(t *testing.T) {
		err := chromedp.Run(ctx,
			chromedp.SetValue(`input[name="teamNumber"]`, "254"),
			chromedp.Click(`form[hx-post*="/admin/makePick"] button[type="submit"]`),
			chromedp.Poll("document.querySelector('#adminMessage').textContent.includes('Successfully picked team 254')", nil, chromedp.WithPollingInterval(100*time.Millisecond)),
			chromedp.OuterHTML("body", &pageBody),
		)
		require.NoError(t, err)
		assert.Contains(t, pageBody, "Successfully picked team 254")
	})

	// Test admin undo pick
	t.Run("admin undo pick works", func(t *testing.T) {
		// Handle the hx-confirm dialog
		chromedp.ListenTarget(ctx, func(ev interface{}) {
			if _, ok := ev.(*page.EventJavascriptDialogOpening); ok {
				go chromedp.Run(ctx, page.HandleJavaScriptDialog(true))
			}
		})
		err := chromedp.Run(ctx,
			chromedp.Click(`button[hx-post*="/admin/undoPick"]`),
			chromedp.Poll("document.querySelector('#adminMessage').textContent.includes('Pick undone successfully')", nil, chromedp.WithPollingInterval(100*time.Millisecond)),
			chromedp.OuterHTML("body", &pageBody),
		)
		require.NoError(t, err)
		assert.Contains(t, pageBody, "Pick undone successfully")
	})

	// Test extend time
	t.Run("admin extend time works", func(t *testing.T) {
		err := chromedp.Run(ctx,
			chromedp.Click(`button[hx-post*="extendTime?duration=30m"]`),
			chromedp.Poll("document.querySelector('#adminMessage').textContent.includes('Pick time extended')", nil, chromedp.WithPollingInterval(100*time.Millisecond)),
			chromedp.OuterHTML("body", &pageBody),
		)
		require.NoError(t, err)
		assert.Contains(t, pageBody, "Pick time extended")
	})
}
