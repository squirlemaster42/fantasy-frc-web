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
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, err
	}
	defer database.CloseStatement(ctx, stmt, "GetTeam")
	team := Team{}
	err = stmt.QueryRowContext(ctx, tbaId).Scan(&team.TbaId, &team.Name, &team.AllianceScore)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	return &team, nil
}

func createTeam(ctx context.Context, db database.DBTX, tbaId string, name string) error {
	query := `INSERT INTO Teams (tbaId, name) Values ($1, $2);`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "CreateTeam")
	_, err = stmt.ExecContext(ctx, tbaId, name)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

func updateTeamAllianceScore(ctx context.Context, db database.DBTX, tbaId string, allianceScore int) error {
	query := `Update Teams Set allianceScore = $1 where tbaId = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "UpdateTeamAllianceScore")
	_, err = stmt.ExecContext(ctx, allianceScore, tbaId)
	return err
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

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, err
	}
	defer database.CloseStatement(ctx, stmt, "GetMatchScores")

	rows, err := stmt.QueryContext(ctx, tbaId)
	if err != nil {
		return nil, fmt.Errorf("failed to get match scores for team: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetMatchScores")

	var matches []MatchTeamScore
	for rows.Next() {
		var match MatchTeamScore
		err := rows.Scan(&match.MatchTbaId, &match.Alliance, &match.Score, &match.IsDqed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan match scores: %w", err)
		}
		matches = append(matches, match)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating matches scores: %w", err)
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

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, err
	}
	defer database.CloseStatement(ctx, stmt, "GetScoresBatch")

	rows, err := stmt.QueryContext(ctx, tbaIds)
	if err != nil {
		return nil, fmt.Errorf("failed to get scores batch: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetScoresBatch")

	for rows.Next() {
		var tbaId string
		var allianceScore, qualScore, playoffScore, einsteinScore, totalScore int
		err = rows.Scan(&tbaId, &allianceScore, &qualScore, &playoffScore, &einsteinScore, &totalScore)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scores batch: %w", err)
		}

		scoresByTeam[tbaId] = map[string]int{
			allianceScoreLabel: allianceScore,
			qualScoreLabel:     qualScore,
			playoffScoreLabel:  playoffScore,
			einsteinScoreLabel: einsteinScore,
			totalScoreLabel:    totalScore,
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating scores batch: %w", err)
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
