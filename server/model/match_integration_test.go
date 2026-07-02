package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddMatchAndGetMatch_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLMatchStore(db)
	ctx := context.Background()

	matchId := "2026cur_qm" + randomString(4)
	err := store.AddMatch(ctx, matchId)
	require.NoError(t, err)

	match, err := store.GetMatch(ctx, matchId)
	require.NoError(t, err)
	require.NotNil(t, match)
	assert.Equal(t, matchId, match.TbaId)
	assert.False(t, match.Played)
	assert.Equal(t, 0, match.RedScore)
	assert.Equal(t, 0, match.BlueScore)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Matches WHERE tbaid = $1", matchId)
	})
}

func TestUpdateScore_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLMatchStore(db)
	ctx := context.Background()

	matchId := "2026cur_qm" + randomString(4)
	err := store.AddMatch(ctx, matchId)
	require.NoError(t, err)

	err = store.UpdateScore(ctx, matchId, 75, 60)
	require.NoError(t, err)

	match, err := store.GetMatch(ctx, matchId)
	require.NoError(t, err)
	assert.True(t, match.Played)
	assert.Equal(t, 75, match.RedScore)
	assert.Equal(t, 60, match.BlueScore)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Matches WHERE tbaid = $1", matchId)
	})
}

func TestAssociateTeam_Integration(t *testing.T) {
	db := setupTestDB(t)
	matchStore := NewSQLMatchStore(db)
	matchTeamStore := NewSQLMatchTeamStore(db)
	teamStore := NewSQLTeamStore(db)
	ctx := context.Background()

	matchId := "2026cur_qm" + randomString(4)
	teamId := "frc" + randomString(4)

	err := matchStore.AddMatch(ctx, matchId)
	require.NoError(t, err)

	err = teamStore.CreateTeam(ctx, teamId, "Test Team")
	require.NoError(t, err)

	err = matchTeamStore.AssociateTeam(ctx, matchId, teamId, "Red", false)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Matches_Teams WHERE match_tbaid = $1", matchId)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Matches WHERE tbaid = $1", matchId)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Teams WHERE tbaId = $1", teamId)
	})
}
