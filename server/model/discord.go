package model

import (
	"context"
	"database/sql"
	"fmt"
	"server/database"
	"server/log"
	"strings"
)

func getPlayerDiscordId(ctx context.Context, db database.DBTX, draftPlayerId int) (sql.NullString, error) {
	query := `
		Select
			u.DiscordId
		From DraftPlayers dp
		Inner Join Users u On u.UserUUID = dp.UserUUID
		Where dp.Id = $1
	`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return sql.NullString{}, err
	}
	defer database.CloseStatement(ctx, stmt, "GetPlayerDiscordId")

	var discordId sql.NullString
	err = stmt.QueryRowContext(ctx, draftPlayerId).Scan(&discordId)
	if err != nil {
		return sql.NullString{}, err
	}

	return discordId, nil
}

// getPlayerPickNotificationId returns the player's Discord ID only if they have opted
// into pick-turn notifications for the draft; otherwise it returns an invalid NullString.
func getPlayerPickNotificationId(ctx context.Context, db database.DBTX, draftPlayerId int) (sql.NullString, error) {
	query := `
		Select
			u.DiscordId,
			COALESCE(p.PickTurn, false) As PickTurn
		From DraftPlayers dp
		Inner Join Users u On u.UserUUID = dp.UserUUID
		Left Join UserDraftNotificationPreferences p
			On p.UserUuid = u.UserUuid And p.DraftId = dp.DraftId
		Where dp.Id = $1
	`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return sql.NullString{}, err
	}
	defer database.CloseStatement(ctx, stmt, "GetPlayerPickNotificationId")

	var discordId sql.NullString
	var wantsPickTurn bool
	err = stmt.QueryRowContext(ctx, draftPlayerId).Scan(&discordId, &wantsPickTurn)
	if err != nil {
		return sql.NullString{}, err
	}

	if !wantsPickTurn {
		return sql.NullString{}, nil
	}

	return discordId, nil
}

func getDraftWebhook(ctx context.Context, db database.DBTX, draftId int) (string, error) {
	query := `
		Select
			d.DiscordWebhook
		From Drafts d
		Where d.Id = $1
	`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return "", err
	}
	defer database.CloseStatement(ctx, stmt, "GetDraftWebhook")

	var webhook sql.NullString
	err = stmt.QueryRowContext(ctx, draftId).Scan(&webhook)
	if err != nil {
		return "", err
	}

	if !webhook.Valid {
		return "", fmt.Errorf("draft with id %d does not have discord webhook set", draftId)
	}

	return webhook.String, nil
}

type DraftPickRow struct {
	DraftId            int
	DraftName          string
	Username           string
	Pick               string
	DiscordId          sql.NullString
	Webhook            sql.NullString
	WantsUpcomingMatch bool
}

func getDraftPickRows(ctx context.Context, db database.DBTX, teamKeys []string) ([]DraftPickRow, error) {
	// Build positional placeholders ($1, $2, ...) and argument slice.
	// Only numeric placeholders are concatenated into the query; all user
	// input values are passed as arguments, so this remains safe from SQL
	// injection.
	placeholders := database.Placeholders(1, len(teamKeys))
	args := make([]interface{}, len(teamKeys))
	for i, key := range teamKeys {
		args[i] = key
	}
	// query
	query := fmt.Sprintf(`
        SELECT
            d.id,
            d.discordwebhook,
            d.displayname,
            u.username,
            u.discordid,
            p.pick,
            COALESCE(prefs.UpcomingMatch, false) AS wants_upcoming_match
        FROM
            Drafts d
        JOIN DraftPlayers dp ON d.id = dp.draftid
        JOIN Users u ON dp.useruuid = u.useruuid
        JOIN Picks p ON dp.id = p.player
        LEFT JOIN UserDraftNotificationPreferences prefs
            ON prefs.UserUuid = u.UserUuid AND prefs.DraftId = d.id
        WHERE
            p.pick IN (%s)
            AND d.discordwebhook IS NOT NULL;
    `, strings.Join(placeholders, ","))
	// prepare query
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		log.Error(ctx, "GetDraftPickRows: Failed to prepare statement", "error", err)
		return nil, err
	}
	defer database.CloseStatement(ctx, stmt, "GetDraftPickRows")

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		log.Error(ctx, "GetDraftPickRows: Failed to execute query", "error", err)
		return nil, err
	}
	defer database.CloseRows(ctx, rows, "GetDraftPickRows")

	var results []DraftPickRow

	for rows.Next() {
		var r DraftPickRow
		err = rows.Scan(&r.DraftId, &r.Webhook, &r.DraftName, &r.Username, &r.DiscordId, &r.Pick, &r.WantsUpcomingMatch)
		if err != nil {
			log.Error(ctx, "GetDraftPickRows: Failed to scan draft pick row", "error", err)
		} else {
			results = append(results, r)
		}
	}

	return results, nil
}
