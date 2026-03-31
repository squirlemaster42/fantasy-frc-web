package model

import (
	"database/sql"
	"server/assert"
	"server/log"
)

func GetPlayerDiscordId(database *sql.DB, draftPlayerId int) (string, error) {
	query := `
		Select
			u.DiscordId
		From DraftPlayers
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

	var discordId string
	err = stmt.QueryRow(draftPlayerId).Scan(&discordId)
	if err != nil {
		return "", err
	}

	return discordId, nil

}

func GetDraftWebhook(database *sql.DB, draftId int) (string, error) {
	query := `
		Select
			d.DiscordWebhook,
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

	var webhook string
	err = stmt.QueryRow(draftId).Scan(&webhook)
	if err != nil {
		return "", err
	}

	return webhook, nil
}
