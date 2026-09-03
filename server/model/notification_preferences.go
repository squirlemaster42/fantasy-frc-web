package model

import (
	"context"
	"fmt"

	"server/database"

	"github.com/google/uuid"
)

// DraftNotificationPreference stores a user's Discord notification opt-ins for a specific draft.
type DraftNotificationPreference struct {
	UserUuid      uuid.UUID
	DraftId       int
	UpcomingMatch bool
	PickTurn      bool
}

// getDraftsForUser returns all drafts the user owns or is an accepted player in.
func getDraftsForUser(ctx context.Context, db database.DBTX, userUuid uuid.UUID) ([]DraftModel, error) {
	query := `
		SELECT DISTINCT
			d.Id,
			d.DisplayName,
			COALESCE(d.Status, '') AS Status
		FROM Drafts d
		LEFT JOIN DraftPlayers dp ON dp.DraftId = d.Id
		WHERE d.OwnerUserUuid = $1
			OR dp.UserUuid = $1
		ORDER BY d.Id DESC
	`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare get drafts for user statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetDraftsForUser")

	rows, err := stmt.QueryContext(ctx, userUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get drafts for user: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetDraftsForUser")

	var drafts []DraftModel
	for rows.Next() {
		var draft DraftModel
		err = rows.Scan(&draft.Id, &draft.DisplayName, &draft.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan draft for user: %w", err)
		}
		drafts = append(drafts, draft)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating drafts for user: %w", err)
	}

	return drafts, nil
}

// getUserDraftNotificationPreferences returns all notification preferences for a user, keyed by DraftId.
func getUserDraftNotificationPreferences(ctx context.Context, db database.DBTX, userUuid uuid.UUID) (map[int]DraftNotificationPreference, error) {
	query := `
		SELECT DraftId, UpcomingMatch, PickTurn
		FROM UserDraftNotificationPreferences
		WHERE UserUuid = $1
	`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare get user notification preferences statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetUserDraftNotificationPreferences")

	rows, err := stmt.QueryContext(ctx, userUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user notification preferences: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetUserDraftNotificationPreferences")

	preferences := make(map[int]DraftNotificationPreference)
	for rows.Next() {
		var pref DraftNotificationPreference
		pref.UserUuid = userUuid
		err = rows.Scan(&pref.DraftId, &pref.UpcomingMatch, &pref.PickTurn)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user notification preference: %w", err)
		}
		preferences[pref.DraftId] = pref
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user notification preferences: %w", err)
	}

	return preferences, nil
}

// updateUserDraftNotificationPreference upserts a user's notification preference for a draft.
func updateUserDraftNotificationPreference(ctx context.Context, db database.DBTX, userUuid uuid.UUID, draftId int, upcomingMatch bool, pickTurn bool) error {
	query := `
		INSERT INTO UserDraftNotificationPreferences (UserUuid, DraftId, UpcomingMatch, PickTurn)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (UserUuid, DraftId)
		DO UPDATE SET UpcomingMatch = EXCLUDED.UpcomingMatch, PickTurn = EXCLUDED.PickTurn
	`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare update user notification preference statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "UpdateUserDraftNotificationPreference")

	_, err = stmt.ExecContext(ctx, userUuid, draftId, upcomingMatch, pickTurn)
	if err != nil {
		return fmt.Errorf("failed to update user notification preference: %w", err)
	}

	return nil
}
