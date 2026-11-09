package model

import (
	"context"
	"database/sql"
	"fmt"
	"server/database"
	"server/log"
	"strings"
)

type Match struct {
	TbaId        string
	Played       bool
	RedScore     int
	BlueScore    int
	RedAlliance  []string
	BlueAlliance []string
	DqedTeams    []string
}

func (m *Match) String() string {
	return fmt.Sprintf("Match: {\nTbaId: %s\n Played: %t\n RedScore: %d\n BlueScore: %d\n RedAlliance: %s\n BlueAlliance: %s\n DqedTeams: %s\n}",
		m.TbaId, m.Played, m.RedScore, m.BlueScore, strings.Join(m.RedAlliance, ", "), strings.Join(m.BlueAlliance, ", "), strings.Join(m.DqedTeams, ", "))
}

func addMatch(ctx context.Context, db database.DBTX, tbaId string) error {
	query := `INSERT INTO Matches (tbaid, played, redscore, bluescore) Values ($1, $2, $3, $4);`
	if _, err := database.Exec(ctx, db, query, tbaId, false, 0, 0); err != nil {
		log.Error(ctx, "Failed to add match", "matchTbaId", tbaId, "error", err)
		return fmt.Errorf("failed to execute addMatch: %w", err)
	}
	return nil
}

func updateScore(ctx context.Context, db database.DBTX, tbaId string, redScore int, blueScore int) error {
	query := `UPDATE Matches Set played = $1, redscore = $2, bluescore = $3 Where tbaid = $4;`
	if _, err := database.Exec(ctx, db, query, true, redScore, blueScore, tbaId); err != nil {
		log.Error(ctx, "Failed to update score", "matchTbaId", tbaId, "redScore", redScore, "blueScore", blueScore, "error", err)
		return fmt.Errorf("failed to execute updateScore: %w", err)
	}
	return nil
}

// All validity checks should be done before now, so we can have this many asserts here
func getMatch(ctx context.Context, db database.DBTX, tbaId string) (*Match, error) {
	query := `Select tbaid, played, redscore, bluescore From Matches Where tbaid = $1;`
	match, err := database.QueryRow(ctx, db, query, func(row *sql.Row) (Match, error) {
		var m Match
		if err := row.Scan(&m.TbaId, &m.Played, &m.RedScore, &m.BlueScore); err != nil {
			log.Error(ctx, "Failed to get match", "matchTbaId", tbaId, "error", err)
			return m, fmt.Errorf("failed to scan getMatch row: %w", err)
		}
		return m, nil
	}, tbaId)
	if err != nil {
		return nil, err
	}
	return &match, nil
}
