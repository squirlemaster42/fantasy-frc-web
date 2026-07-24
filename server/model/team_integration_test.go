package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTeamAndGetTeam_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLTeamStore(db)
	ctx := context.Background()

	teamId := "frc" + randomString(4)
	err := store.CreateTeam(ctx, teamId, "Test Team")
	require.NoError(t, err)

	team, err := store.GetTeam(ctx, teamId)
	require.NoError(t, err)
	require.NotNil(t, team)
	assert.Equal(t, teamId, team.TbaId)
	assert.Equal(t, "Test Team", team.Name)
	assert.Equal(t, 0, team.AllianceScore)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Teams WHERE tbaId = $1", teamId)
	})
}

func TestUpdateTeamAllianceScore_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLTeamStore(db)
	ctx := context.Background()

	teamId := "frc" + randomString(4)
	err := store.CreateTeam(ctx, teamId, "Test Team")
	require.NoError(t, err)

	err = store.UpdateTeamAllianceScore(ctx, teamId, 25)
	require.NoError(t, err)

	team, err := store.GetTeam(ctx, teamId)
	require.NoError(t, err)
	assert.Equal(t, 25, team.AllianceScore)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Teams WHERE tbaId = $1", teamId)
	})
}

func TestGetScore_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLTeamStore(db)
	ctx := context.Background()

	teamId := "frc" + randomString(4)
	err := store.CreateTeam(ctx, teamId, "Test Team")
	require.NoError(t, err)

	err = store.UpdateTeamAllianceScore(ctx, teamId, 10)
	require.NoError(t, err)

	scores, err := store.GetScore(ctx, teamId)
	require.NoError(t, err)
	assert.Equal(t, 10, scores["Alliance Score"])
	assert.Equal(t, 10, scores["Total Score"])

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Teams WHERE tbaId = $1", teamId)
	})
}

func TestGetMatchScores_Integration(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLTeamStore(db)
	ctx := context.Background()

	teamId := "frc" + randomString(4)
	err := store.CreateTeam(ctx, teamId, "Test Team")
	require.NoError(t, err)

	matchId := "2026cur_qm1"
	_, err = db.ExecContext(ctx, "INSERT INTO Matches (tbaid, played, redscore, bluescore) VALUES ($1, true, 50, 30) ON CONFLICT (tbaid) DO UPDATE SET played = EXCLUDED.played, redscore = EXCLUDED.redscore, bluescore = EXCLUDED.bluescore", matchId)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, "INSERT INTO Matches_Teams (match_tbaid, team_tbaid, alliance, isdqed) VALUES ($1, $2, 'Red', false) ON CONFLICT DO NOTHING", matchId, teamId)
	require.NoError(t, err)

	matches, err := store.GetMatchScores(ctx, teamId)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, matchId, matches[0].MatchTbaId)
	assert.Equal(t, "Red", matches[0].Alliance)
	assert.Equal(t, 50, matches[0].Score)
	assert.False(t, matches[0].IsDqed)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Matches_Teams WHERE team_tbaid = $1", teamId)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Matches WHERE tbaid = $1", matchId)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM Teams WHERE tbaId = $1", teamId)
	})
}
