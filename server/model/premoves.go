package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"server/database"
)

const (
	premoveStatusPending = "Pending"
	premoveStatusMade    = "Made"

	// PickOrder is NOT NULL and unique per draft player, so in-place reordering
	// would collide mid-transaction. Rows are parked in these reserved positive
	// ranges while the queue is rebuilt. Each player has at most PicksPerPlayer
	// pending premoves and that many made ones, far below the offsets.
	premoveMadeOrderBase = 10000
	premoveParkOrderBase = 20000
)

// ErrTeamNotAvailable indicates that a premove references a team that is not
// present in the Teams table and therefore cannot be picked.
var ErrTeamNotAvailable = errors.New("team is not available")

// TeamNotAvailableError identifies the specific team that cannot be used as a
// premove.
type TeamNotAvailableError struct {
	TeamTbaId string
}

func (e *TeamNotAvailableError) Error() string {
	return fmt.Sprintf("team %s is not available", e.TeamTbaId)
}

func (e *TeamNotAvailableError) Unwrap() error {
	return ErrTeamNotAvailable
}

type Premove struct {
	Id            int
	TeamTbaId     string
	Status        string
	Error         string
	PickOrder     int
	DraftPlayerId int
}

// IsPending reports whether the premove is still waiting to be made.
func (p Premove) IsPending() bool {
	return p.Status == premoveStatusPending || p.Status == ""
}

// IsMade reports whether the premove has already been used to make a pick.
func (p Premove) IsMade() bool {
	return p.Status == premoveStatusMade
}

// premoveScanner is satisfied by both *sql.Row and *sql.Rows.
type premoveScanner interface {
	Scan(dest ...any) error
}

func scanPremove(scanner premoveScanner) (Premove, error) {
	var premove Premove
	err := scanner.Scan(
		&premove.Id,
		&premove.TeamTbaId,
		&premove.Status,
		&premove.Error,
		&premove.PickOrder,
		&premove.DraftPlayerId,
	)
	if err != nil {
		return Premove{}, err
	}
	return premove, nil
}

func hasPremove(ctx context.Context, db database.DBTX, draftPlayerId int) (bool, error) {
	query := `SELECT EXISTS (
		SELECT 1 FROM PickMoves pm
		WHERE pm.DraftPlayerId = $1
			AND (pm.Status = $2 OR pm.Status IS NULL)
			AND NOT EXISTS (
				SELECT 1
				FROM Picks p
				INNER JOIN DraftPlayers pickedPlayer ON pickedPlayer.Id = p.Player
				INNER JOIN DraftPlayers owner ON owner.Id = pm.DraftPlayerId
				WHERE p.Pick = pm.TeamTBAId
					AND pickedPlayer.DraftId = owner.DraftId
			)
	);`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return false, err
	}
	defer database.CloseStatement(ctx, stmt, "HasPremove")

	var has bool
	err = stmt.QueryRowContext(ctx, draftPlayerId, premoveStatusPending).Scan(&has)
	if err != nil {
		return false, fmt.Errorf("failed to check premove existence: %w", err)
	}

	return has, nil
}

func getPremoves(ctx context.Context, db database.DBTX, draftPlayerId int) ([]Premove, error) {
	query := `SELECT
		Id,
		COALESCE(TeamTBAId, ''),
		COALESCE(Status::text, ''),
		COALESCE(Error, ''),
		COALESCE(PickOrder, 0),
		COALESCE(DraftPlayerId, 0)
	FROM PickMoves
	WHERE DraftPlayerId = $1
	ORDER BY PickOrder ASC NULLS LAST, Id ASC;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, err
	}
	defer database.CloseStatement(ctx, stmt, "GetPremoves")

	rows, err := stmt.QueryContext(ctx, draftPlayerId)
	if err != nil {
		return nil, fmt.Errorf("failed to query premoves: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetPremoves")

	var premoves []Premove
	for rows.Next() {
		premove, err := scanPremove(rows)
		if err != nil {
			return nil, err
		}
		premoves = append(premoves, premove)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate premoves: %w", err)
	}

	return premoves, nil
}

func getPremove(ctx context.Context, db database.DBTX, draftPlayerId int) (Premove, error) {
	query := `SELECT
		pm.Id,
		COALESCE(pm.TeamTBAId, ''),
		COALESCE(pm.Status::text, ''),
		COALESCE(pm.Error, ''),
		COALESCE(pm.PickOrder, 0),
		COALESCE(pm.DraftPlayerId, 0)
	FROM PickMoves pm
	WHERE pm.DraftPlayerId = $1
		AND (pm.Status = $2 OR pm.Status IS NULL)
		AND NOT EXISTS (
			SELECT 1
			FROM Picks p
			INNER JOIN DraftPlayers pickedPlayer ON pickedPlayer.Id = p.Player
			INNER JOIN DraftPlayers owner ON owner.Id = pm.DraftPlayerId
			WHERE p.Pick = pm.TeamTBAId
				AND pickedPlayer.DraftId = owner.DraftId
		)
	ORDER BY pm.PickOrder ASC NULLS LAST, pm.Id ASC
	LIMIT 1;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return Premove{}, err
	}
	defer database.CloseStatement(ctx, stmt, "GetPremove")

	premove, err := scanPremove(stmt.QueryRowContext(ctx, draftPlayerId, premoveStatusPending))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Premove{}, sql.ErrNoRows
		}
		return Premove{}, err
	}

	return premove, nil
}

func addPremove(ctx context.Context, db database.DBTX, draftPlayerId int, teamTbaId string, pickOrder int) error {
	if draftPlayerId <= 0 {
		return fmt.Errorf("invalid draft player id: %d", draftPlayerId)
	}
	if teamTbaId == "" {
		return fmt.Errorf("team tba id cannot be empty")
	}

	query := `INSERT INTO PickMoves (DraftPlayerId, TeamTBAId, Status, PickOrder)
		SELECT $1, tbaId, $3, $4 FROM Teams WHERE tbaId = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "AddPremove")

	result, err := stmt.ExecContext(ctx, draftPlayerId, teamTbaId, premoveStatusPending, pickOrder)
	if err != nil {
		return fmt.Errorf("failed to add premove: %w", err)
	}

	if err := requirePremoveInserted(result, teamTbaId); err != nil {
		return err
	}

	return nil
}

func requirePremoveInserted(result sql.Result, teamTbaId string) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected adding premove: %w", err)
	}
	if rowsAffected == 0 {
		return &TeamNotAvailableError{TeamTbaId: teamTbaId}
	}
	return nil
}

// replacePremoves inserts newly queued teams, updates the pick order of teams
// that are already pending, and removes pending premoves that are no longer in
// the queue. Teams that do not exist in the Teams table are skipped and
// returned so callers can warn the player without discarding the rest of the
// queue. Callers must run this inside a transaction so a failed change cannot
// leave the player with a partially updated queue.
func replacePremoves(ctx context.Context, db database.DBTX, draftPlayerId int, teamTbaIds []string) ([]string, error) {
	if draftPlayerId <= 0 {
		return nil, fmt.Errorf("invalid draft player id: %d", draftPlayerId)
	}

	desiredOrder := make(map[string]int, len(teamTbaIds))
	for i, teamTbaId := range teamTbaIds {
		if teamTbaId == "" {
			return nil, fmt.Errorf("team tba id cannot be empty")
		}
		if _, exists := desiredOrder[teamTbaId]; exists {
			return nil, fmt.Errorf("duplicate team tba id: %s", teamTbaId)
		}
		desiredOrder[teamTbaId] = i + 1
	}

	premoves, err := getPremoves(ctx, db, draftPlayerId)
	if err != nil {
		return nil, err
	}

	pendingByTeam := make(map[string]Premove, len(premoves))
	completedTeams := make(map[string]struct{}, len(premoves))
	for _, premove := range premoves {
		if premove.IsPending() {
			pendingByTeam[premove.TeamTbaId] = premove
			continue
		}
		completedTeams[premove.TeamTbaId] = struct{}{}
	}

	// Move every order for the player into a reserved range before assigning the
	// new ones. Reordering or reusing an order that is still held by another
	// premove would otherwise violate the unique (DraftPlayerId, PickOrder)
	// index mid-transaction.
	parkQuery := `UPDATE PickMoves p
		SET PickOrder = ($2 + parked.rn)::smallint
		FROM (
			SELECT Id, row_number() OVER (ORDER BY Id) AS rn
			FROM PickMoves
			WHERE DraftPlayerId = $1
		) AS parked
		WHERE p.Id = parked.Id;`
	parkStmt, err := database.Prepare(ctx, db, parkQuery)
	if err != nil {
		return nil, err
	}
	defer database.CloseStatement(ctx, parkStmt, "ReplacePremoves")

	if _, err := parkStmt.ExecContext(ctx, draftPlayerId, premoveParkOrderBase); err != nil {
		return nil, fmt.Errorf("failed to park premove orders: %w", err)
	}

	// Keep completed premoves out of the pending order range. Ordering by Id
	// preserves the order in which the premoves were originally created.
	madeQuery := `UPDATE PickMoves p
		SET PickOrder = ($2 + parked.rn)::smallint
		FROM (
			SELECT Id, row_number() OVER (ORDER BY Id) AS rn
			FROM PickMoves
			WHERE DraftPlayerId = $1 AND Status <> $3
		) AS parked
		WHERE p.Id = parked.Id;`
	madeStmt, err := database.Prepare(ctx, db, madeQuery)
	if err != nil {
		return nil, err
	}
	defer database.CloseStatement(ctx, madeStmt, "ReplacePremoves")

	if _, err := madeStmt.ExecContext(ctx, draftPlayerId, premoveMadeOrderBase, premoveStatusPending); err != nil {
		return nil, fmt.Errorf("failed to reorder completed premoves: %w", err)
	}

	var updateStmt *sql.Stmt
	defer func() { database.CloseStatement(ctx, updateStmt, "ReplacePremoves") }()

	var insertStmt *sql.Stmt
	defer func() { database.CloseStatement(ctx, insertStmt, "ReplacePremoves") }()

	unavailable := make([]string, 0)
	nextOrder := 1
	for _, teamTbaId := range teamTbaIds {
		if _, completed := completedTeams[teamTbaId]; completed {
			continue
		}

		premove, exists := pendingByTeam[teamTbaId]
		if exists {
			if updateStmt == nil {
				updateQuery := `UPDATE PickMoves SET PickOrder = $1
					WHERE Id = $2 AND (Status = $3 OR Status IS NULL);`
				updateStmt, err = database.Prepare(ctx, db, updateQuery)
				if err != nil {
					return nil, err
				}
			}

			if _, err := updateStmt.ExecContext(ctx, nextOrder, premove.Id, premoveStatusPending); err != nil {
				return nil, fmt.Errorf("failed to update premove order for %s: %w", teamTbaId, err)
			}
			nextOrder++
			continue
		}

		if insertStmt == nil {
			insertQuery := `INSERT INTO PickMoves (DraftPlayerId, TeamTBAId, Status, PickOrder)
				SELECT $1, tbaId, $3, $4 FROM Teams WHERE tbaId = $2;`
			insertStmt, err = database.Prepare(ctx, db, insertQuery)
			if err != nil {
				return nil, err
			}
		}

		result, err := insertStmt.ExecContext(ctx, draftPlayerId, teamTbaId, premoveStatusPending, nextOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to add premove %s: %w", teamTbaId, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to get rows affected adding premove %s: %w", teamTbaId, err)
		}
		if rowsAffected == 0 {
			unavailable = append(unavailable, teamTbaId)
			continue
		}
		nextOrder++
	}

	var deleteStmt *sql.Stmt
	defer func() { database.CloseStatement(ctx, deleteStmt, "ReplacePremoves") }()

	removedTeams := make(map[string]struct{}, len(pendingByTeam))
	for _, premove := range premoves {
		if !premove.IsPending() {
			continue
		}
		if _, keep := desiredOrder[premove.TeamTbaId]; keep {
			continue
		}
		if _, removed := removedTeams[premove.TeamTbaId]; removed {
			continue
		}
		removedTeams[premove.TeamTbaId] = struct{}{}

		if deleteStmt == nil {
			deleteQuery := `DELETE FROM PickMoves
				WHERE DraftPlayerId = $1 AND TeamTBAId = $2 AND (Status = $3 OR Status IS NULL);`
			deleteStmt, err = database.Prepare(ctx, db, deleteQuery)
			if err != nil {
				return nil, err
			}
		}

		if _, err := deleteStmt.ExecContext(ctx, draftPlayerId, premove.TeamTbaId, premoveStatusPending); err != nil {
			return nil, fmt.Errorf("failed to remove premove %s: %w", premove.TeamTbaId, err)
		}
	}

	return unavailable, nil
}

func markPremoveComplete(ctx context.Context, db database.DBTX, premoveId int) error {
	query := `UPDATE PickMoves SET Status = $1 WHERE Id = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "MarkPremoveComplete")

	result, err := stmt.ExecContext(ctx, premoveStatusMade, premoveId)
	if err != nil {
		return fmt.Errorf("failed to mark premove complete: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected marking premove complete: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("premove %d not found", premoveId)
	}

	return nil
}

func restorePremoveToTop(ctx context.Context, db database.DBTX, draftPlayerId int, teamTbaId string) error {
	if draftPlayerId <= 0 {
		return fmt.Errorf("invalid draft player id: %d", draftPlayerId)
	}
	if teamTbaId == "" {
		return fmt.Errorf("team tba id cannot be empty")
	}

	parkQuery := `UPDATE PickMoves p
		SET PickOrder = ($2 + parked.rn)::smallint
		FROM (
			SELECT Id, row_number() OVER (ORDER BY Id) AS rn
			FROM PickMoves
			WHERE DraftPlayerId = $1
		) AS parked
		WHERE p.Id = parked.Id;`
	parkStmt, err := database.Prepare(ctx, db, parkQuery)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, parkStmt, "RestorePremoveToTop")
	if _, err := parkStmt.ExecContext(ctx, draftPlayerId, premoveParkOrderBase); err != nil {
		return fmt.Errorf("failed to park premove orders: %w", err)
	}

	madeQuery := `UPDATE PickMoves p
		SET PickOrder = ($2 + parked.rn)::smallint
		FROM (
			SELECT Id, row_number() OVER (ORDER BY Id) AS rn
			FROM PickMoves
			WHERE DraftPlayerId = $1 AND Status <> $3
		) AS parked
		WHERE p.Id = parked.Id;`
	madeStmt, err := database.Prepare(ctx, db, madeQuery)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, madeStmt, "RestorePremoveToTop")
	if _, err := madeStmt.ExecContext(ctx, draftPlayerId, premoveMadeOrderBase, premoveStatusPending); err != nil {
		return fmt.Errorf("failed to reorder completed premoves: %w", err)
	}

	updateQuery := `UPDATE PickMoves SET Status = $4, PickOrder = 1, Error = NULL
		WHERE DraftPlayerId = $1 AND TeamTBAId = $2 AND Status = $3;`
	updateStmt, err := database.Prepare(ctx, db, updateQuery)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, updateStmt, "RestorePremoveToTop")

	result, err := updateStmt.ExecContext(ctx, draftPlayerId, teamTbaId, premoveStatusMade, premoveStatusPending)
	if err != nil {
		return fmt.Errorf("failed to restore premove: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected restoring premove: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("premove for team %s not found for draft player %d", teamTbaId, draftPlayerId)
	}

	rebuildQuery := `UPDATE PickMoves p
		SET PickOrder = rebuild.new_order
		FROM (
			SELECT Id,
				row_number() OVER (ORDER BY CASE WHEN TeamTBAId = $3 THEN 0 ELSE 1 END, PickOrder ASC NULLS LAST, Id ASC) AS new_order
			FROM PickMoves
			WHERE DraftPlayerId = $1 AND (Status = $2 OR Status IS NULL)
		) AS rebuild
		WHERE p.Id = rebuild.Id;`
	rebuildStmt, err := database.Prepare(ctx, db, rebuildQuery)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, rebuildStmt, "RestorePremoveToTop")

	if _, err := rebuildStmt.ExecContext(ctx, draftPlayerId, premoveStatusPending, teamTbaId); err != nil {
		return fmt.Errorf("failed to shift premove queue: %w", err)
	}

	return nil
}

