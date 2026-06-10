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

func TestE2E_ViewInvitesPage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	createTestUser(t, ts, "inviteuser", "Password123")

	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login and navigate to invites page
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "inviteuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/viewInvites"),
		chromedp.WaitVisible(`#pendingTable`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("invites page shows empty state", func(t *testing.T) {
		assert.Contains(t, pageBody, "No Pending Invitations")
		assert.Contains(t, pageBody, "You'll see draft invitations here when someone invites you to join.")
	})
}

func TestE2E_AcceptInvite(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx := context.Background()

	// Create owner and invitee
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "owner", "Password123")
	inviteeUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "invitee", "Password123")

	// Create a draft and invite the user directly via DB
	draftId := createTestDraft(t, ts, ownerUuid, "Invite Draft", "For invite testing")
	inviteId, err := ts.Handler.DraftStore.InvitePlayer(ctx, draftId, ownerUuid, inviteeUuid)
	require.NoError(t, err)

	ctx2, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login as invitee and navigate to invites
	err = chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "invitee"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
		chromedp.Navigate(ts.BaseURL+"/u/viewInvites"),
		chromedp.WaitVisible(fmt.Sprintf(`button[value="%d"]`, inviteId)),
		chromedp.Click(fmt.Sprintf(`button[value="%d"]`, inviteId)),
		chromedp.Poll(`document.querySelector('#pendingTable').innerText.includes('No Pending Invitations')`, nil, chromedp.WithPollingInterval(100*time.Millisecond), chromedp.WithPollingTimeout(5*time.Second)),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("invite accepted successfully", func(t *testing.T) {
		assert.Contains(t, pageBody, "No Pending Invitations")
	})

	// Verify the user is now in the draft
	 draftModel, err := ts.Handler.DraftStore.GetDraft(ctx, draftId)
	require.NoError(t, err)
	found := false
	for _, p := range draftModel.Players {
		if p.User.UserUuid == inviteeUuid {
			found = true
			break
		}
	}
	assert.True(t, found, "invitee should be in the draft after accepting")
}
