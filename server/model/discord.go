package model

import (
	"database/sql"
	"fmt"
	"log/slog"
	"server/assert"
	"server/log"
	"strings"
)

func GetPlayerDiscordId(database *sql.DB, draftPlayerId int) (sql.NullString, error) {
	query := `
		Select
			u.DiscordId
		From DraftPlayers dp
		Inner Join Users u On u.UserUUID = dp.UserUUID
		Where dp.Id = $1
	`

	assert := assert.CreateAssertWithContext("Get Player Discord Id")
	assert.AddContext("Draft Player Id", draftPlayerId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.WarnNoContext("CreateDraft: Failed to close statement", "error", err)
		}
	}()

	var discordId sql.NullString
	err = stmt.QueryRow(draftPlayerId).Scan(&discordId)
	if err != nil {
		return sql.NullString{}, err
	}

	return discordId, nil
}

func GetDraftWebhook(database *sql.DB, draftId int) (string, error) {
	query := `
		Select
			d.DiscordWebhook
		From Drafts d
		Where d.Id = $1
	`

	assert := assert.CreateAssertWithContext("Get Next Pick Discord Event")
	assert.AddContext("Draft Id", draftId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.WarnNoContext("CreateDraft: Failed to close statement", "error", err)
		}
	}()

	var webhook sql.NullString
	err = stmt.QueryRow(draftId).Scan(&webhook)
	if err != nil {
		return "", err
	}

	if !webhook.Valid {
		return "", fmt.Errorf("Draft with id %d does not have discord webhook set", draftId)
	}

	return webhook.String, nil
}

type DraftPickRow struct {
	DraftId   int
	DraftName string
	Username  string
	Pick      string
	DiscordId sql.NullString
	Webhook   sql.NullString
}

func GetDraftPickRows(database *sql.DB, teamKeys []string) ([]DraftPickRow, error) {
	// set up query params
	placeholders := make([]string, len(teamKeys))
	args := make([]interface{}, len(teamKeys))
	for i, key := range teamKeys {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
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
            p.pick
        FROM 
            Drafts d
        JOIN DraftPlayers dp ON d.id = dp.draftid
        JOIN Users u ON dp.useruuid = u.useruuid
        JOIN Picks p ON dp.id = p.player
        WHERE 
            p.pick IN (%s)
            AND d.discordwebhook IS NOT NULL;
    `, strings.Join(placeholders, ","))
	// prepare query
	stmt, err := database.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DraftPickRow

	for rows.Next() {
		var r DraftPickRow
		err = rows.Scan(&r.DraftId, &r.Webhook, &r.DraftName, &r.Username, &r.DiscordId, &r.Pick)
		if err != nil {
			slog.Warn("Failed to scan draft pick row")
		} else {
			results = append(results, r)
		}
	}

	return results, nil
}
