package view

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"

	"server/model"
)

func TestHomeIndex_EmptyState(t *testing.T) {
	drafts := []model.DraftModel{}
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	var buf strings.Builder
	err := HomeIndex(drafts, userUuid).Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	doc, err := html.Parse(strings.NewReader(htmlStr))
	require.NoError(t, err)

	t.Run("empty state message", func(t *testing.T) {
		assert.Contains(t, htmlStr, "No Drafts Yet")
		assert.Contains(t, htmlStr, "Create your first fantasy draft to get started with your friends.")
	})

	t.Run("create draft link", func(t *testing.T) {
		createLink := findElementByAttr(doc, "a", "href", "/u/createDraft")
		require.NotNil(t, createLink)
		assert.Contains(t, textContent(createLink), "Create New Draft")
	})

	t.Run("no draft cards", func(t *testing.T) {
		cards := findAllElementsByTag(doc, "div")
		for _, card := range cards {
			if hasAttrContaining(card, "class", "card bg-base-200") {
				// Should not have draft-specific cards in empty state
				assert.NotContains(t, getAttr(card, "class"), "shadow-lg")
			}
		}
	})
}

func TestHomeIndex_WithDrafts(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	otherUserUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")

	drafts := []model.DraftModel{
		{
			Id:          1,
			DisplayName: "Test Draft",
			Description: "A test draft",
			Status:      model.FILLING,
			Owner:       model.User{UserUuid: userUuid, Username: "owneruser"},
			NextPick:    model.DraftPlayer{User: model.User{Username: "nextpicker"}},
			Players: []model.DraftPlayer{
				{User: model.User{Username: "player1"}, Pending: false},
				{User: model.User{Username: "player2"}, Pending: true},
			},
		},
		{
			Id:          2,
			DisplayName: "Another Draft",
			Status:      model.PICKING,
			Owner:       model.User{UserUuid: otherUserUuid, Username: "otherowner"},
			NextPick:    model.DraftPlayer{User: model.User{Username: ""}},
			Players: []model.DraftPlayer{
				{User: model.User{Username: "player3"}, Pending: false},
			},
		},
	}

	var buf strings.Builder
	err := HomeIndex(drafts, userUuid).Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	doc, err := html.Parse(strings.NewReader(htmlStr))
	require.NoError(t, err)

	t.Run("draft cards present", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Test Draft")
		assert.Contains(t, htmlStr, "Another Draft")
	})

	t.Run("owner badge on first draft", func(t *testing.T) {
		// The first draft is owned by userUuid, so it should have a star icon
		assert.Contains(t, htmlStr, `title="Owner"`)
	})

	t.Run("status badges", func(t *testing.T) {
		assert.Contains(t, htmlStr, string(model.FILLING))
		assert.Contains(t, htmlStr, string(model.PICKING))
	})

	t.Run("next pick displays", func(t *testing.T) {
		assert.Contains(t, htmlStr, "nextpicker")
	})

	t.Run("next pick N/A for empty", func(t *testing.T) {
		assert.Contains(t, htmlStr, "N/A")
	})

	t.Run("joined players", func(t *testing.T) {
		assert.Contains(t, htmlStr, "player1")
		assert.Contains(t, htmlStr, "player3")
	})

	t.Run("pending players", func(t *testing.T) {
		assert.Contains(t, htmlStr, "player2")
	})

	t.Run("open draft links", func(t *testing.T) {
		openLink := findElementByAttr(doc, "a", "href", "/u/draft/1/profile")
		require.NotNil(t, openLink)
		assert.Contains(t, textContent(openLink), "Open")
	})

	t.Run("create new draft card", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Create New Draft")
	})

	t.Run("search input present", func(t *testing.T) {
		searchInput := findElementByAttr(doc, "input", "type", "search")
		require.NotNil(t, searchInput)
		assert.Contains(t, getAttr(searchInput, "class"), "bg-base-200/50")
		assert.Equal(t, "/u/draftList", getAttr(searchInput, "hx-get"))
		assert.Equal(t, "#draft-list", getAttr(searchInput, "hx-target"))
		assert.Equal(t, "innerHTML", getAttr(searchInput, "hx-swap"))
	})

	t.Run("draft list container present", func(t *testing.T) {
		draftList := findElementByAttr(doc, "div", "id", "draft-list")
		require.NotNil(t, draftList)
		assert.Contains(t, getAttr(draftList, "class"), "grid")
	})

	t.Run("create new draft card is direct child of draft list grid", func(t *testing.T) {
		draftList := findElementByAttr(doc, "div", "id", "draft-list")
		require.NotNil(t, draftList)
		found := false
		for c := draftList.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && hasAttrContaining(c, "class", "card bg-gradient-to-br") {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

func TestDraftSearchResults_EmptySearchMessage(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	searchTerm := "nonexistent"

	var buf strings.Builder
	err := DraftSearchResults([]model.DraftModel{}, userUuid, 0, searchTerm).Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	doc, err := html.Parse(strings.NewReader(htmlStr))
	require.NoError(t, err)

	assert.Contains(t, htmlStr, "No Drafts Found")
	assert.Contains(t, htmlStr, searchTerm)
	assert.NotContains(t, htmlStr, "Create New Draft</h2>")
	createLink := findElementByAttr(doc, "a", "href", "/u/createDraft")
	require.NotNil(t, createLink)
}

func TestDraftList_DoesNotShowEmptySearchMessage(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	searchTerm := "nonexistent"

	var buf strings.Builder
	err := DraftList([]model.DraftModel{}, userUuid, 1, searchTerm).Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.NotContains(t, htmlStr, "No drafts found matching")
}

func TestHome_PageWrapper(t *testing.T) {
	var buf strings.Builder
	err := Home("Home", true, "testuser", HomeIndex([]model.DraftModel{}, uuid.New())).Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.Contains(t, htmlStr, "<!doctype html>")
	assert.Contains(t, htmlStr, `hx-boost="true"`)
	assert.Contains(t, htmlStr, "testuser")
}
