package draft

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"server/model"
	"server/types"
)

func TestDraftPickIndex(t *testing.T) {
	pickPage := PickPage{
		Draft: model.DraftModel{
			Id: 42,
			Players: []model.DraftPlayer{
				{
					Id: 1,
					User: model.User{Username: "player1"},
					Picks: []model.Pick{
						{Pick: sql.NullString{Valid: true, String: "frc254"}, PickTime: sql.NullTime{Valid: true, Time: time.Now()}},
						{Skipped: true},
					},
				},
				{
					Id: 2,
					User: model.User{Username: "player2"},
					Picks: []model.Pick{
						{ExpirationTime: time.Now().Add(1 * time.Hour)},
					},
				},
			},
			NextPick: model.DraftPlayer{Id: 2, User: model.User{Username: "player2"}},
		},
		PickUrl:       "/u/draft/42/makePick",
		NotifierUrl:   "/u/draft/42/pickNotifier",
		IsCurrentPick: true,
		IsSkipping:    false,
		SkipUrl:       "/u/draft/42/skipPickToggle",
	}

	var buf strings.Builder
	err := DraftPickIndex(pickPage, "csrf").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()

	t.Run("websocket connection", func(t *testing.T) {
		assert.Contains(t, htmlStr, `hx-ext="ws"`)
		assert.Contains(t, htmlStr, `ws-connect="/u/draft/42/pickNotifier"`)
	})

	t.Run("form hx-post", func(t *testing.T) {
		assert.Contains(t, htmlStr, `hx-post="/u/draft/42/makePick"`)
		assert.Contains(t, htmlStr, `hx-swap="outerHTML transition:true"`)
		assert.Contains(t, htmlStr, `hx-target="#draftPicks"`)
	})

	t.Run("draft pick header", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Draft Picks")
	})

	t.Run("make pick button", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Make Pick")
		assert.Contains(t, htmlStr, `type="submit"`)
		assert.Contains(t, htmlStr, `hx-disabled-elt="this"`)
	})

	t.Run("skip picks checkbox", func(t *testing.T) {
		assert.Contains(t, htmlStr, `id="skip-picks-checkbox"`)
		assert.Contains(t, htmlStr, `name="skipping"`)
		assert.Contains(t, htmlStr, `type="checkbox"`)
		assert.Contains(t, htmlStr, `hx-post="/u/draft/42/skipPickToggle"`)
	})

	t.Run("alpine loading state", func(t *testing.T) {
		assert.Contains(t, htmlStr, `x-data="{ loading: false, status: '' }"`)
		assert.Contains(t, htmlStr, `@htmx:before-request`)
		assert.Contains(t, htmlStr, `@htmx:after-request`)
		assert.Contains(t, htmlStr, `@htmx:response-error`)
	})

	t.Run("csrf token", func(t *testing.T) {
		assert.Contains(t, htmlStr, `name="csrf_token"`)
	})

	t.Run("rendered picks", func(t *testing.T) {
		assert.Contains(t, htmlStr, "player1")
		assert.Contains(t, htmlStr, "player2")
		assert.Contains(t, htmlStr, "254")
		assert.Contains(t, htmlStr, "Skipped")
	})

	t.Run("pick input for next pick", func(t *testing.T) {
		assert.Contains(t, htmlStr, `name="pickInput"`)
	})
}

func TestDraftPickIndex_WithError(t *testing.T) {
	pickPage := PickPage{
		Draft: model.DraftModel{
			Id:      42,
			Players: []model.DraftPlayer{},
			NextPick: model.DraftPlayer{},
		},
		PickUrl:       "/u/draft/42/makePick",
		NotifierUrl:   "/u/draft/42/pickNotifier",
		IsCurrentPick: false,
		IsSkipping:    true,
		PickError:     errors.New("Invalid team number"),
		SkipUrl:       "/u/draft/42/skipPickToggle",
	}

	var buf strings.Builder
	err := DraftPickIndex(pickPage, "csrf").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()

	t.Run("error message displayed", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Invalid team number")
		assert.Contains(t, htmlStr, `id="pickError"`)
	})

	t.Run("skip checkbox checked", func(t *testing.T) {
		assert.Contains(t, htmlStr, `checked`)
	})
}

func TestDraftPick_PageWrapper(t *testing.T) {
	var buf strings.Builder
	pickPage := PickPage{
		Draft: model.DraftModel{Id: 42},
	}
	err := DraftPick("Draft Picks", true, "testuser", DraftPickIndex(pickPage, "csrf"), types.NewPageData(42, "Test Draft", true)).Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.Contains(t, htmlStr, "<!doctype html>")
	assert.Contains(t, htmlStr, `hx-boost="true"`)
	assert.Contains(t, htmlStr, "testuser")
}

func TestRenderPicks(t *testing.T) {
	draft := model.DraftModel{
		Players: []model.DraftPlayer{
			{
				Id:   1,
				User: model.User{Username: "player1"},
				Picks: []model.Pick{
								{Pick: sql.NullString{Valid: true, String: "frc254"}, PickTime: sql.NullTime{Valid: true, Time: time.Now()}},
								{Skipped: true},
								{Pick: sql.NullString{Valid: true, String: "frc118"}, PickTime: sql.NullTime{Valid: true, Time: time.Now()}},
				},
			},
		},
		NextPick: model.DraftPlayer{Id: 1, User: model.User{Username: "player1"}},
	}

	var buf strings.Builder
	err := RenderPicks(draft, true).Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()

	t.Run("pick table id", func(t *testing.T) {
		assert.Contains(t, htmlStr, `id="pickTable"`)
		assert.Contains(t, htmlStr, `hx-swap="outerHTML"`)
	})

	t.Run("player name", func(t *testing.T) {
		assert.Contains(t, htmlStr, "player1")
	})

	t.Run("picked teams", func(t *testing.T) {
		assert.Contains(t, htmlStr, "254")
		assert.Contains(t, htmlStr, "118")
	})

	t.Run("skipped pick", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Skipped")
	})

	t.Run("input for current pick", func(t *testing.T) {
		assert.Contains(t, htmlStr, `name="pickInput"`)
	})
}
