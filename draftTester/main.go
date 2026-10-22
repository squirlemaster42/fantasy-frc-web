package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"math/rand/v2"
	"time"

	"draftclient"
)

const target = "http://localhost:7331"

type rawUser struct {
	Username string `json:"Username"`
	Password string `json:"Password"`
}

func main() {
	client := draftclient.New(target)

	userJSON, err := client.LoadUserConfig("./localUserConfig.json")
	if err != nil {
		log.Fatal(err)
	}

	var rawUsers []rawUser
	if err := json.Unmarshal([]byte(userJSON), &rawUsers); err != nil {
		log.Fatal(err)
	}

	users := make([]*draftclient.User, len(rawUsers))
	for i, ru := range rawUsers {
		users[i] = client.CreateUser(ru.Username, ru.Password)
	}

	for _, user := range users {
		client.Login(user)
	}

	owner := users[rand.IntN(len(users))]
	owner.IsOwner = true

	draft := client.CreateDraft(owner)
	client.InvitePlayersToDraft(owner, users, draft)
	for _, user := range users {
		client.AcceptInvite(user)
	}
	client.StartDraft(owner, draft.Id)
	client.WaitUntilDraftState(owner, draft.Id, "Picking", 300*time.Second)

	slog.Info("Starting to make picks")
	validTeams := client.LoadValidTeams("./frc-worlds-2026.csv")

	successfulPicks := 0
	invalidPicks := 0

	for client.GetCurrentDraftStatus(owner, draft.Id) != "Teams Playing" {
		var pickingPlayer *draftclient.User
		for _, player := range users {
			if client.IsPickingPlayer(player, draft.Id) {
				pickingPlayer = player
				break
			}
		}
		if pickingPlayer == nil {
			panic("failed to find picking player")
		}

		slog.Info("Got picking player", "Username", pickingPlayer.Username)
		pickingPlayer = users[rand.IntN(len(users))]
		slog.Info("Chose random player instead", "Username", pickingPlayer.Username)

		success, _ := client.MakePickRequest(draft.Id, pickingPlayer, getRandomTeamId(validTeams))
		if success {
			successfulPicks++
		} else {
			invalidPicks++
		}

		if successfulPicks > 64 {
			panic("too many picks were made")
		}
		slog.Info("Picking round made", "Successful Picks", successfulPicks, "Invalid Picks", invalidPicks)
	}
}

func getRandomTeamId(validPicks []int) int {
	return validPicks[rand.IntN(len(validPicks))]
}
