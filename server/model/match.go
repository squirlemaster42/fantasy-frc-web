package model

import (
	"context"
	"database/sql"
	"fmt"
	db "server/database"
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

func addMatch(ctx context.Context, database *sql.DB, tbaId string) error {
	query := `INSERT INTO Matches (tbaid, played, redscore, bluescore) Values ($1, $2, $3, $4);`
	stmt, err := db.Prepare(ctx, database, query)
	if err != nil {
		return err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "AddMatch: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, tbaId, false, 0, 0)
	if err != nil {
		log.Error(ctx, "Failed to add match", "matchTbaId", tbaId, "error", err)
		return err
	}
	return nil
}

func updateScore(ctx context.Context, database *sql.DB, tbaId string, redScore int, blueScore int) error {
	query := `UPDATE Matches Set played = $1, redscore = $2, bluescore = $3 Where tbaid = $4;`
	stmt, err := db.Prepare(ctx, database, query)
	if err != nil {
		return err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "UpdateScore: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, true, redScore, blueScore, tbaId)
	if err != nil {
		log.Error(ctx, "Failed to update score", "matchTbaId", tbaId, "redScore", redScore, "blueScore", blueScore, "error", err)
		return err
	}
	return nil
}

// All validity checks should be done before now, so we can have this many asserts here
func getMatch(ctx context.Context, database *sql.DB, tbaId string) (*Match, error) {
	query := `Select tbaid, played, redscore, bluescore From Matches Where tbaid = $1;`
	stmt, err := db.Prepare(ctx, database, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetMatch: Failed to close statement", "error", err)
		}
	}()
	match := Match{}
	err = stmt.QueryRowContext(ctx, tbaId).Scan(&match.TbaId, &match.Played, &match.RedScore, &match.BlueScore)
	if err != nil {
		log.Error(ctx, "Failed to get match", "matchTbaId", tbaId, "error", err)
		return nil, err
	}
	return &match, nil
}
