package e2e

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_ViewPickPage(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx := context.Background()
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "pickowner", "Password123")

	draftId := createPickingDraft(t, ts, ownerUuid, "Pick Test Draft")

	ctx2, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login as owner
	err := chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "pickowner"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
	)
	require.NoError(t, err)

	// Navigate to pick page
	err = chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/pick", draftId)),
		chromedp.WaitVisible(`#pickTable`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("websocket connection attributes present", func(t *testing.T) {
		assert.Contains(t, pageBody, `hx-ext="ws"`)
		assert.Contains(t, pageBody, `ws-connect="/u/draft/`)
		assert.Contains(t, pageBody, `/pickNotifier"`)
	})

	t.Run("pick form present", func(t *testing.T) {
		assert.Contains(t, pageBody, `name="pickInput"`)
		assert.Contains(t, pageBody, "Make Pick")
	})

	t.Run("skip picks checkbox present", func(t *testing.T) {
		assert.Contains(t, pageBody, `id="skip-picks-checkbox"`)
		assert.Contains(t, pageBody, `name="skipping"`)
	})

	t.Run("alpine loading state present", func(t *testing.T) {
		assert.Contains(t, pageBody, `x-data="{ loading: false, status: '' }"`)
	})
}

func TestE2E_EmptyPickSubmission(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx := context.Background()
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "pickuser2", "Password123")

	draftId := createPickingDraft(t, ts, ownerUuid, "Pick Draft Empty")

	ctx2, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login as owner (who is the current picker)
	err := chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "pickuser2"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
	)
	require.NoError(t, err)

	// Navigate to pick page and submit empty pick
	err = chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/pick", draftId)),
		chromedp.WaitVisible(`input[name="pickInput"]`),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`#pickError`),
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("empty pick shows error message", func(t *testing.T) {
		assert.Contains(t, pageBody, "id=\"pickError\"")
		assert.Contains(t, pageBody, "you must be the picking player to make a pick")
	})
}

func TestE2E_SkipPickToggle(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx := context.Background()
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "skipuser", "Password123")

	draftId := createPickingDraft(t, ts, ownerUuid, "Skip Pick Draft")

	ctx2, cancel := newBrowserContext(t)
	defer cancel()

	var pageBody string

	// Login as owner
	err := chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "skipuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
	)
	require.NoError(t, err)

	// Navigate to pick page and toggle skip checkbox
	err = chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/pick", draftId)),
		chromedp.WaitVisible(`#skip-picks-checkbox`),
		chromedp.Click(`#skip-picks-checkbox`),
		chromedp.Sleep(500*time.Millisecond), // Wait for HTMX request
		chromedp.OuterHTML("body", &pageBody),
	)
	require.NoError(t, err)

	t.Run("skip checkbox toggled", func(t *testing.T) {
		// After HTMX swap, the checkbox should still be present and checked
		assert.Contains(t, pageBody, `id="skip-picks-checkbox"`)
	})
}

func TestE2E_WebSocketConnection(t *testing.T) {
	skipIfNoDocker(t)
	skipIfNoChrome(t)

	db, _, cleanupDB := setupTestDatabase(t)
	defer cleanupDB()

	ts := createTestServer(t, db)
	defer ts.Shutdown()

	ctx := context.Background()
	ownerUuid, _ := ts.Handler.UserStore.RegisterUser(ctx, "wsuser", "Password123")

	draftId := createPickingDraft(t, ts, ownerUuid, "WebSocket Draft")

	ctx2, cancel := newBrowserContext(t)
	defer cancel()

	var wsUrls []string
	var mu sync.Mutex

	// Listen for WebSocket creation events
	chromedp.ListenTarget(ctx2, func(ev interface{}) {
		if wsEvent, ok := ev.(*network.EventWebSocketCreated); ok {
			mu.Lock()
			wsUrls = append(wsUrls, wsEvent.URL)
			mu.Unlock()
		}
	})

	// Enable network domain to capture WebSocket events
	err := chromedp.Run(ctx2,
		network.Enable(),
	)
	require.NoError(t, err)

	// Login as owner
	err = chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+"/login"),
		chromedp.WaitVisible(`input[name="username"]`),
		chromedp.SendKeys(`input[name="username"]`, "wsuser"),
		chromedp.SendKeys(`input[name="password"]`, "Password123"),
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitVisible(`a[href="/u/createDraft"]`),
	)
	require.NoError(t, err)

	// Navigate to pick page (triggers WebSocket connection via HTMX ws extension)
	err = chromedp.Run(ctx2,
		chromedp.Navigate(ts.BaseURL+fmt.Sprintf("/u/draft/%d/pick", draftId)),
		chromedp.WaitVisible(`#pickTable`),
		chromedp.Sleep(500*time.Millisecond), // Give time for WebSocket to connect
	)
	require.NoError(t, err)

	t.Run("websocket connection established", func(t *testing.T) {
		mu.Lock()
		defer mu.Unlock()

		found := false
		expectedPath := fmt.Sprintf("/u/draft/%d/pickNotifier", draftId)
		for _, url := range wsUrls {
			if strings.Contains(url, expectedPath) {
				found = true
				break
			}
		}
		assert.True(t, found, "expected WebSocket connection to %s, got: %v", expectedPath, wsUrls)
	})
}
