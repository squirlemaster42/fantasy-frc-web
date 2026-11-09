package model

import (
	"context"
	"database/sql"
	"fmt"
	"server/database"

	"uuid"
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

	return database.QueryRows(ctx, db, query, func(rows *sql.Rows) (DraftModel, error) {
		var draft DraftModel
		if err := rows.Scan(&draft.Id, &draft.DisplayName, &draft.Status); err != nil {
			return draft, fmt.Errorf("failed to scan draft for user: %w", err)
		}
		return draft, nil
	}, userUuid)
}

// getUserDraftNotificationPreferences returns all notification preferences for a user, keyed by DraftId.
func getUserDraftNotificationPreferences(ctx context.Context, db database.DBTX, userUuid uuid.UUID) (map[int]DraftNotificationPreference, error) {
	query := `
		SELECT DraftId, UpcomingMatch, PickTurn
		FROM UserDraftNotificationPreferences
		WHERE UserUuid = $1
	`

	prefs, err := database.QueryRows(ctx, db, query, func(rows *sql.Rows) (DraftNotificationPreference, error) {
		var pref DraftNotificationPreference
		pref.UserUuid = userUuid
		if err := rows.Scan(&pref.DraftId, &pref.UpcomingMatch, &pref.PickTurn); err != nil {
			return pref, fmt.Errorf("failed to scan user notification preference: %w", err)
		}
		return pref, nil
	}, userUuid)
	if err != nil {
		return nil, err
	}

	preferences := make(map[int]DraftNotificationPreference, len(prefs))
	for _, pref := range prefs {
		preferences[pref.DraftId] = pref
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

	if _, err := database.Exec(ctx, db, query, userUuid, draftId, upcomingMatch, pickTurn); err != nil {
		return fmt.Errorf("failed to update user notification preference: %w", err)
	}
	return nil
}
