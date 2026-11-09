package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/database"
	"server/log"
	"server/utils"
	"sort"
)

type Team struct {
	TbaId         string
	Name          string
	AllianceScore int
}

func (t *Team) String() string {
	return fmt.Sprintf("Team: {\n TbaId: %s\n Name: %s\n AllianceScore: %d\n}",
		t.TbaId, t.Name, t.AllianceScore)
}

func getTeam(ctx context.Context, db database.DBTX, tbaId string) (*Team, error) {
	query := `Select tbaId, name, COALESCE(allianceScore, 0) As allianceScore From Teams Where tbaId = $1;`
	team, err := database.QueryRow(ctx, db, query, func(row *sql.Row) (Team, error) {
		var t Team
		if err := row.Scan(&t.TbaId, &t.Name, &t.AllianceScore); err != nil {
			return t, fmt.Errorf("failed to get team: %w", err)
		}
		return t, nil
	}, tbaId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // not found is represented as nil team with nil error
		}
		return nil, err
	}
	return &team, nil
}

func createTeam(ctx context.Context, db database.DBTX, tbaId string, name string) error {
	query := `INSERT INTO Teams (tbaId, name) Values ($1, $2);`
	if _, err := database.Exec(ctx, db, query, tbaId, name); err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

func updateTeamAllianceScore(ctx context.Context, db database.DBTX, tbaId string, allianceScore int) error {
	query := `Update Teams Set allianceScore = $1 where tbaId = $2;`
	if _, err := database.Exec(ctx, db, query, allianceScore, tbaId); err != nil {
		return fmt.Errorf("failed to execute updateTeamAllianceScore: %w", err)
	}
	return nil
}

// MatchTeamScore represents a team's score in a specific match
type MatchTeamScore struct {
	MatchTbaId string
	Alliance   string // "Red" or "Blue"
	Score      int
	IsDqed     bool
}

// GetQualificationReturns individual qualification match scores for a team
func getMatchScores(ctx context.Context, db database.DBTX, tbaId string) ([]MatchTeamScore, error) {
	query := `
		Select
			mt.Match_tbaId,
			mt.Alliance,
			Case When mt.Alliance = 'Red' then m.redscore When mt.Alliance = 'Blue' Then m.bluescore Else 0 End As Score,
			mt.Isdqed
		From Matches_Teams mt
		Inner Join Matches m On mt.Match_tbaId = m.tbaId
		Where mt.Team_TbaId = $1
		Order By mt.Match_tbaId`

	matches, err := database.QueryRows(ctx, db, query, func(rows *sql.Rows) (MatchTeamScore, error) {
		var match MatchTeamScore
		if err := rows.Scan(&match.MatchTbaId, &match.Alliance, &match.Score, &match.IsDqed); err != nil {
			return match, fmt.Errorf("failed to scan match scores: %w", err)
		}
		return match, nil
	}, tbaId)
	if err != nil {
		return nil, err
	}

	sort.Slice(matches, func(i, j int) bool {
		val, err := utils.CompareMatchOrder(ctx, matches[i].MatchTbaId, matches[j].MatchTbaId)
		if err != nil {
			log.Error(ctx, "Failed to compare matches", "match1", matches[i].MatchTbaId, "match2", matches[j].MatchTbaId, "error", err)
		}
		return val
	})

	return matches, nil
}

// getScoresBatch returns the score breakdown for many teams in a single query.
// The inner map keys are: Alliance Score, Qual Score, Playoff Score, Einstein Score, Total Score.
func getScoresBatch(ctx context.Context, db database.DBTX, tbaIds []string) (map[string]map[string]int, error) {
	scoresByTeam := make(map[string]map[string]int)
	if len(tbaIds) == 0 {
		return scoresByTeam, nil
	}

	query := `Select
		t.TbaId,
		COALESCE(t.AllianceScore, 0) AS "Alliance Score",
		COALESCE(SUM(CASE WHEN mt.match_tbaId Like '%_qm%' THEN
			CASE WHEN mt.Alliance = 'Red' THEN m.redscore
			     WHEN mt.Alliance = 'Blue' THEN m.bluescore
			     ELSE 0 END
			ELSE 0 END), 0) AS "Qual Score",
		COALESCE(SUM(CASE WHEN mt.match_tbaId Not Like '%_qm%' AND mt.match_tbaId Not Like '%cmptx%' THEN
			CASE WHEN mt.Alliance = 'Red' THEN m.redscore
			     WHEN mt.Alliance = 'Blue' THEN m.bluescore
			     ELSE 0 END
			ELSE 0 END), 0) AS "Playoff Score",
		COALESCE(SUM(CASE WHEN mt.match_tbaId Like '%cmptx%' THEN
			CASE WHEN mt.Alliance = 'Red' THEN m.redscore
			     WHEN mt.Alliance = 'Blue' THEN m.bluescore
			     ELSE 0 END
			ELSE 0 END), 0) AS "Einstein Score",
		COALESCE(t.AllianceScore, 0) +
		COALESCE(SUM(CASE WHEN mt.Alliance = 'Red' THEN m.redscore
			     WHEN mt.Alliance = 'Blue' THEN m.bluescore
			     ELSE 0 END), 0) AS "Total Score"
	From Teams t
	Left Join Matches_Teams mt On mt.Team_TbaId = t.TbaId And mt.Isdqed = false
	Left Join Matches m On mt.Match_tbaId = m.tbaId
	Where t.TbaId = ANY($1)
	Group By t.TbaId, t.AllianceScore
	Order By t.TbaId`

	rows, err := database.QueryRows(ctx, db, query, func(rows *sql.Rows) (struct {
		tbaId                                    string
		allianceScore, qualScore, playoffScore int
		einsteinScore, totalScore                int
	}, error) {
		var row struct {
			tbaId                                    string
			allianceScore, qualScore, playoffScore int
			einsteinScore, totalScore                int
		}
		if err := rows.Scan(&row.tbaId, &row.allianceScore, &row.qualScore, &row.playoffScore, &row.einsteinScore, &row.totalScore); err != nil {
			return row, fmt.Errorf("failed to scan scores batch: %w", err)
		}
		return row, nil
	}, tbaIds)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		scoresByTeam[row.tbaId] = map[string]int{
			allianceScoreLabel: row.allianceScore,
			qualScoreLabel:     row.qualScore,
			playoffScoreLabel:  row.playoffScore,
			einsteinScoreLabel: row.einsteinScore,
			totalScoreLabel:    row.totalScore,
		}
	}

	return scoresByTeam, nil
}

// Keys are the string that represents display name and the value is the score
// for that display name
// Display names: Qual Score, Playoff Score, Alliance Score, Einstein Score, Total Score
func getScore(ctx context.Context, db database.DBTX, tbaId string) (map[string]int, error) {
	scores, err := getScoresBatch(ctx, db, []string{tbaId})
	if err != nil {
		return nil, err
	}

	teamScores, ok := scores[tbaId]
	if !ok {
		return nil, fmt.Errorf("failed to get score for team: %w", sql.ErrNoRows)
	}

	log.Debug(ctx, "Got scores for team", "team", tbaId, "scores", teamScores)

	return teamScores, nil
}
