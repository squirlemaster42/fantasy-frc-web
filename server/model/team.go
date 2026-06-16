package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/assert"
	"server/log"
	"server/tbaHandler"
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

func getTeam(ctx context.Context, database *sql.DB, tbaId string) (*Team, error) {
	query := `Select tbaId, name, COALESCE(allianceScore, 0) As allianceScore From Teams Where tbaId = $1;`
	assert := assert.CreateAssertWithContext("Get Team")
	assert.AddContext("TbaId", tbaId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetTeam: Failed to close statement", "error", err)
		}
	}()
	team := Team{}
	err = stmt.QueryRowContext(ctx, tbaId).Scan(&team.TbaId, &team.Name, &team.AllianceScore)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	if team.TbaId == "" {
		return nil, nil
	}
	return &team, nil
}

func createTeam(ctx context.Context, database *sql.DB, tbaId string, name string) error {
	query := `INSERT INTO Teams (tbaId, name) Values ($1, $2);`
	assert := assert.CreateAssertWithContext("Create Team")
	assert.AddContext("Tba Id", tbaId)
	assert.AddContext("Name", name)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "CreateTeam: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, tbaId, name)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

func updateTeamAllianceScore(ctx context.Context, database *sql.DB, tbaId string, allianceScore int16) error {
	query := `Update Teams Set allianceScore = $1 where tbaId = $2;`
	assert := assert.CreateAssertWithContext("Update Team Alliance Score")
	assert.AddContext("Tba Id", tbaId)
	assert.AddContext("Alliance Score", allianceScore)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "UpdateTeamAllianceScore: Failed to close statement", "error", err)
		}
	}()
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
func getMatchScores(ctx context.Context, database *sql.DB, tbaId string) ([]MatchTeamScore, error) {
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

	assert := assert.CreateAssertWithContext("Get Qualification Matches")
	assert.AddContext("TbaId", tbaId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetMatchScores: Failed to close statement", "error", err)
		}
	}()

	rows, err := stmt.QueryContext(ctx, tbaId)
	if err != nil {
		return nil, fmt.Errorf("failed to get match scores for team: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Warn(ctx, "GetMatchScores: Failed to close rows", "error", err)
		}
	}()

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
			log.Warn(ctx, "Failed to compare matches", "Match 1", matches[i].MatchTbaId, "Match 2", matches[j].MatchTbaId, "Error", err)
		}
		return val
	})

	return matches, nil
}

func ValidPick(ctx context.Context, draftStore DraftStore, teamStore TeamStore, handler *tbaHandler.TBAHandler, tbaId string, draftId int) (bool, error) {
	if tbaId == "" {
		return false, errors.New("no team entered")
	}

	picked, err := draftStore.HasBeenPicked(ctx, draftId, tbaId)
	if err != nil {
		return false, err
	}

	if picked {
		return false, errors.New("team already picked")
	}

	events := handler.MakeEventListReq(ctx, tbaId)
	draftEvents := utils.Events()

	validEvent := false
	//Looping here should always be faster because of the small lists
	log.Info(ctx, "Checking is team is in a valid event", "Team Events", events, "Draft Events", draftEvents)
	for _, event := range events {
		for _, draftEvent := range draftEvents {
			if event == draftEvent {
				validEvent = true
				break
			}
		}

		if validEvent {
			break
		}
	}

	log.Info(ctx, "Checked if team is a valid pick", "Team", tbaId, "Picked", picked, "Valid Event", validEvent)
	if !validEvent {
		return false, errors.New("team not at event")
	}

	return true, nil
}

// Keys are the string that represents display name and the value is the score
// for that display name
// Display names: Qual Score, Playoff Score, Alliance Score, Einstein Score, Total Score
func getScore(ctx context.Context, database *sql.DB, tbaId string) (map[string]int, error) {
	query := `Select
                COALESCE(t.AllianceScore, 0) As AllianceScore
            From Teams t
            Where t.TbaId = $1`

	assert := assert.CreateAssertWithContext("Get Score")
	assert.AddContext("TbaId", tbaId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetScore: Failed to close statement", "error", err)
		}
	}()

	var allianceScore int
	err = stmt.QueryRowContext(ctx, tbaId).Scan(&allianceScore)
	if err != nil {
		return nil, fmt.Errorf("failed to get alliance score for team: %w", err)
	}

	query = `Select
                Case When mt.match_tbaId Like '%_qm%' Then 'Qual Score'
                     When mt.match_tbaId Like '%cmptx%' Then 'Einstein Score'
                     Else 'Playoff Score' End As DisplayName,
                Sum(Case When mt.Alliance = 'Red' then m.redscore When mt.Alliance = 'Blue' Then m.bluescore Else 0 End) As Score
             From Matches_Teams mt
             Inner Join Matches m On mt.Match_tbaId = m.tbaId
             Where mt.Team_TbaId = $1
             And mt.Isdqed = false
             Group By mt.Team_TbaId, Case When mt.match_tbaId Like '%_qm%' Then 'Qual Score'
                     When mt.match_tbaId Like '%cmptx%' Then 'Einstein Score'
                     Else 'Playoff Score' End
             Order By mt.Team_TbaId`

	stmt, err = database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetScore: Failed to close statement", "error", err)
		}
	}()

	var displayName string
	var matchScore int
	rows, err := stmt.QueryContext(ctx, tbaId)
	if err != nil {
		return nil, fmt.Errorf("failed to get score for team: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Warn(ctx, "GetScore: Failed to close rows", "error", err)
		}
	}()

	scores := make(map[string]int)
	total := 0
	for rows.Next() {
		err = rows.Scan(&displayName, &matchScore)

		if err != nil {
			return nil, fmt.Errorf("failed to scan scores for team: %w", err)
		}

		total += matchScore
		scores[displayName] = matchScore
	}

	scores["Alliance Score"] = allianceScore
	scores["Total Score"] = total + allianceScore

	log.Info(ctx, "Got scores for team", "Team", tbaId, "Scores", scores)

	return scores, nil
}
