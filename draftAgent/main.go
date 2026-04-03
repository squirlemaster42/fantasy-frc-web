package main

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"os/exec"
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

		pick := 0
		slog.Info("Got picking player", "Username", pick)

		makePickRequest(draft.Id, pickingPlayer, 0)
		slog.Info("Picking round made", "Team", pick)
	}
}

func requestNextDraftPick(pickingPlayer *User, sameSession bool) (int, error) {
	var flags []string
	if sameSession {
		flags = append(flags, "-c")
	}

	// TODO Get current picks
	prompt := fmt.Sprintf(
		"%s %s Your name is %s. The current picks in the draft for each player are %s.",
		SystemPrompt,
		pickingPlayer.Persona.PersonaPrompt,
		pickingPlayer.Username,
		"",
	)

	resp, err := callOpencode(prompt, flags...)
	if err != nil {
		return -1, err
	}
	fmt.Println(resp)

	return 0, nil
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
