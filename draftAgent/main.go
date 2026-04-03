package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os/exec"
	"strconv"
)

const  (
	SystemPrompt = `Respond with a single team number. Do not respond with anything else. Do not give an explanation for your pick, just give the team number.`
)

type DrafterPersona struct {
	Model string `json:"Model"`
	PersonaPrompt string `json:"PersonaPrompt"`
}

func main() {
	users, err := initUsers("./userConfig.json")
	if err != nil {
		log.Fatal(err)
	}

	owner, draft := initDraft(users)

	// Have play make picks in a random order. Some picks being valid and some being invalid
	sameSession := false
	additionalPrompt := ""
	for getCurrentDraftStatus(owner, draft.Id) != "Teams Playing" {
		var pickingPlayer *User
		for _, player := range users {
			if isPickingPlayer(player, draft.Id) {
				pickingPlayer = player
				break

			}
		}

		if pickingPlayer == nil {
			panic("failed to find picking player")
		}

		nextPick, err := requestNextDraftPick(pickingPlayer, draft.Id, sameSession, additionalPrompt)
		if err != nil {
			additionalPrompt = "Make sure your pick is only a single team number and contains no additional reasoning"
			continue
		}

		slog.Info("Got picking player", "Username", pickingPlayer.Username, "Pick", nextPick)

		makePickRequest(draft.Id, pickingPlayer, nextPick)
		slog.Info("Picking round made", "Team", nextPick)
	}
}

func requestNextDraftPick(pickingPlayer *User, draftId int, sameSession bool, additionalPrompt string) (int, error) {
	var flags []string
	if sameSession {
		flags = append(flags, "-c")
	}

	currentPicks, err := getCurrentDraftPicks(pickingPlayer, draftId)
	if err != nil {
		return -1, nil
	}

	json, err := json.Marshal(currentPicks)
	if err != nil {
		return -1, err
	}

	prompt := fmt.Sprintf(
		"%s %s Your name is %s. The current picks in the draft for each player are %s. %s",
		SystemPrompt,
		pickingPlayer.Persona.PersonaPrompt,
		pickingPlayer.Username,
		json,
		additionalPrompt,
	)

	resp, err := callOpencode(prompt, flags...)
	if err != nil {
		return -1, err
	}
	fmt.Println(resp)

	return strconv.Atoi(resp)
}

func callOpencode(prompt string, flags ...string) (string, error) {
	var args []string
	args = append(args, "run")
	args = append(args, flags...)
	args = append(args, prompt)
	cmd := exec.Command("opencode", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("opencode failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), err
}
