package main

import (
	"fmt"
	"log"
	"log/slog"
	"time"

	"draftclient"
)

const target = "https://fantasy-frc.cfh.sh"

func main() {
	agentUsers, client, err := loadAgentUserConfig("./userConfig.json")
	if err != nil {
		log.Fatal(err)
	}

	draft := draftclient.Draft{
		Id: 36,
	}

	var owner *draftclient.User
	for _, user := range agentUsers {
		if user.User.Username == "AgentOne" {
			owner = user.User
		}
	}

	slog.Info("Created draft", "Id", draft.Id)

	sameSession := false
	additionalPrompt := ""
	for client.GetCurrentDraftStatus(owner, draft.Id) != "Teams Playing" {
		var pickingPlayer *AgentUser
		for _, player := range agentUsers {
			if client.IsPickingPlayer(player.User, draft.Id) {
				pickingPlayer = player
				break
			}
		}

		if pickingPlayer == nil {
			slog.Warn("failed to find picking player")
			time.Sleep(2 * time.Minute)
			continue
		}
		slog.Info("Picking Player", "Player", pickingPlayer.User.Username)

		slog.Info("Starting pick round", "Picking player", pickingPlayer.User.Username, "Continue", sameSession)

		nextPick, err := requestNextDraftPick(client, pickingPlayer, draft.Id, sameSession, additionalPrompt)
		if err != nil {
			slog.Info("Opencode pick was not a number", "Pick", nextPick, "Error", err)
			additionalPrompt = "Make sure your pick is only a single team number and contains no additional reasoning"
			sameSession = true
			continue
		}

		slog.Info("Got picking player", "Username", pickingPlayer.User.Username, "Pick", nextPick)

		pickMade, errMsg := client.MakePickRequest(draft.Id, pickingPlayer.User, nextPick)
		if !pickMade {
			slog.Error("Pick failed", "Error", errMsg)
			additionalPrompt = fmt.Sprintf("The previous pick was invalid. We got the following error message from the server: %s. Make sure that your team has not been picked yet and is at the 2026 FIRST World Championship. You can use The Blue Alliance website to figure out what teams are at the event.", errMsg)
			sameSession = true
			continue
		}
		slog.Info("Picking round made", "Team", nextPick)
		sameSession = false
		additionalPrompt = ""
	}
}
