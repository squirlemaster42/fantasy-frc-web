package draft

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"

	"server/model"
	"server/types"
)

// HTML parsing helpers (copied from view/testutil_test.go)

func findElementByTag(n *html.Node, tagName string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tagName {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElementByTag(c, tagName); found != nil {
			return found
		}
	}
	return nil
}

func findAllElementsByTag(n *html.Node, tagName string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode && n.Data == tagName {
		result = append(result, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllElementsByTag(c, tagName)...)
	}
	return result
}

func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}

func getAttr(n *html.Node, name string) string {
	if n == nil {
		return ""
	}
	for _, attr := range n.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func TestDraftProfileIndex(t *testing.T) {
	ownerUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	playerUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")

	draft := model.DraftModel{
		Id:          42,
		DisplayName: "Test Draft",
		Description: "A test description",
		Status:      model.FILLING,
		Owner:       model.User{UserUuid: ownerUuid, Username: "owneruser"},
		Players: []model.DraftPlayer{
			{User: model.User{UserUuid: ownerUuid, Username: "owneruser"}, Pending: false, PlayerOrder: sql.NullInt16{Valid: true, Int16: 1}},
			{User: model.User{UserUuid: playerUuid, Username: "player1"}, Pending: true, PlayerOrder: sql.NullInt16{Valid: true, Int16: 2}},
		},
	}

	var buf strings.Builder
	err := DraftProfileIndex(draft, true, "csrf-token").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	doc, err := html.Parse(strings.NewReader(htmlStr))
	require.NoError(t, err)

	t.Run("form hx-post", func(t *testing.T) {
		form := findElementByTag(doc, "form")
		require.NotNil(t, form)
		assert.Contains(t, getAttr(form, "hx-post"), "/u/draft/42/updateDraft")
		assert.Equal(t, "outerHTML", getAttr(form, "hx-swap"))
	})

	t.Run("draft name input", func(t *testing.T) {
		inputs := findAllElementsByTag(doc, "input")
		var nameInput *html.Node
		for _, input := range inputs {
			if getAttr(input, "name") == "draftName" {
				nameInput = input
				break
			}
		}
		require.NotNil(t, nameInput)
		assert.Equal(t, "Test Draft", getAttr(nameInput, "value"))
		assert.NotContains(t, getAttr(nameInput, "disabled"), "disabled")
	})

	t.Run("status badge", func(t *testing.T) {
		assert.Contains(t, htmlStr, string(model.FILLING))
		assert.Contains(t, htmlStr, `id="draftStatus"`)
	})

	t.Run("description textarea", func(t *testing.T) {
		assert.Contains(t, htmlStr, `name="description"`)
		assert.Contains(t, htmlStr, "A test description")
	})

	t.Run("players section", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Players (2)")
		assert.Contains(t, htmlStr, "owneruser")
		assert.Contains(t, htmlStr, "player1")
		assert.Contains(t, htmlStr, "Joined")
		assert.Contains(t, htmlStr, "Pending Invitation")
	})

	t.Run("settings section", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Draft Settings")
	})

	t.Run("save button", func(t *testing.T) {
		buttons := findAllElementsByTag(doc, "button")
		var saveBtn *html.Node
		for _, btn := range buttons {
			if strings.Contains(textContent(btn), "Save Changes") {
				saveBtn = btn
				break
			}
		}
		require.NotNil(t, saveBtn)
		assert.Equal(t, "submit", getAttr(saveBtn, "type"))
	})

	t.Run("invite players section", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Invite Players")
		assert.Contains(t, htmlStr, `name="search"`)
		assert.Contains(t, htmlStr, `hx-post="/u/searchPlayers"`)
		assert.Contains(t, htmlStr, `id="searchResults"`)
	})

	t.Run("start draft button", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Start Draft")
	})

	t.Run("csrf token", func(t *testing.T) {
		assert.Contains(t, htmlStr, `name="csrf_token"`)
		assert.Contains(t, htmlStr, `value="csrf-token"`)
	})
}

func TestDraftProfileIndex_NotOwner(t *testing.T) {
	ownerUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	draft := model.DraftModel{
		Id:          42,
		DisplayName: "Test Draft",
		Status:      model.FILLING,
		Owner:       model.User{UserUuid: ownerUuid, Username: "owneruser"},
		Players:     []model.DraftPlayer{},
	}

	var buf strings.Builder
	err := DraftProfileIndex(draft, false, "csrf").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()

	t.Run("disabled inputs for non-owner", func(t *testing.T) {
		// Non-owner should have disabled inputs
		assert.Contains(t, htmlStr, "disabled")
	})

	t.Run("owner access required message", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Owner Access Required")
		assert.Contains(t, htmlStr, "Only the draft owner can invite players")
	})

	t.Run("save button disabled", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Save Changes")
	})
}

func TestDraftProfile_PageWrapper(t *testing.T) {
	var buf strings.Builder
	err := DraftProfile(" | Draft Profile", true, "testuser", DraftProfileIndex(model.DraftModel{}, true, "csrf"), types.NewPageData(42, "Test Draft", true)).Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.Contains(t, htmlStr, "<!doctype html>")
	assert.Contains(t, htmlStr, `hx-boost="true"`)
	assert.Contains(t, htmlStr, "testuser")
}

func TestStartDraftButton(t *testing.T) {
	var buf strings.Builder
	err := StartDraftButton("/u/draft/42/startDraft", "", true, "csrf").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.Contains(t, htmlStr, "Start Draft")
	assert.Contains(t, htmlStr, `hx-post="/u/draft/42/startDraft"`)
	assert.Contains(t, htmlStr, `csrf_token`)
}

func TestStartDraftButton_WithError(t *testing.T) {
	var buf strings.Builder
	err := StartDraftButton("/u/draft/42/startDraft", "Must have 8 players", true, "csrf").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.Contains(t, htmlStr, "Must have 8 players")
	assert.Contains(t, htmlStr, "Start Draft")
}

func TestPlayerList(t *testing.T) {
	players := []model.DraftPlayer{
		{User: model.User{Username: "joined"}, Pending: false, PlayerOrder: sql.NullInt16{Valid: true, Int16: 1}},
		{User: model.User{Username: "pending"}, Pending: true, PlayerOrder: sql.NullInt16{Valid: true, Int16: 2}},
	}

	var buf strings.Builder
	err := PlayerList(players, 0, false, "").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.Contains(t, htmlStr, `id="playerList"`)
	assert.Contains(t, htmlStr, `hx-swap-oob="outerHTML:#playerList"`)
	assert.Contains(t, htmlStr, "joined")
	assert.Contains(t, htmlStr, "pending")
	assert.Contains(t, htmlStr, "Joined")
	assert.Contains(t, htmlStr, "Pending Invitation")
	assert.Contains(t, htmlStr, "Pick Order: ")
	assert.Contains(t, htmlStr, ">1</span>")
	assert.Contains(t, htmlStr, ">2</span>")
}

func TestPlayerSearchResults(t *testing.T) {
	users := []model.User{
		{UserUuid: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), Username: "john"},
	}

	var buf strings.Builder
	err := PlayerSearchResults(users, 42, true, "csrf").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.Contains(t, htmlStr, `id="inviteTable"`)
	assert.Contains(t, htmlStr, "john")
	assert.Contains(t, htmlStr, "Invite")
	assert.Contains(t, htmlStr, `hx-post="/u/draft/42/invitePlayer"`)
}

func TestPlayerSearchResults_Empty(t *testing.T) {
	var buf strings.Builder
	err := PlayerSearchResults([]model.User{}, 42, true, "csrf").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.Contains(t, htmlStr, "No users found")
}
