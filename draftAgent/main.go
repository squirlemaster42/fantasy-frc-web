package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	SystemPrompt = `Respond with a single team number. Do not respond with anything else. Do not give an explanation for your pick, just give the team number.

You must not pick a team that has already been drafted. You must not pick a team that is not competing at the 2026 FIRST World Championship. Verify a team's attendance at the 2026 FIRST World Championship using The Blue Alliance (thebluealliance.com).

You may use the following resources on the common internet to research team performance, statistics, event history, and current standings:
- The Blue Alliance (thebluealliance.com) — comprehensive team profiles, event results, match data, and rankings
- Statbotics (statbotics.io) — advanced statistics, EPA ratings, Elo ratings, and performance predictions
- Chief Delphi (chiefdelphi.com) — FRC community forums, scouting discussions, and team analysis threads
- FRC Events (frc-events.firstinspires.org) — official FIRST event registrations, schedules, and live results`
)

type DrafterPersona struct {
	Model         string `json:"Model"`
	Variant       string `json:"Variant"`
	PersonaPrompt string `json:"PersonaPrompt"`
}

func main() {
	users, err := initUsers("./userConfig.json")
	if err != nil {
		log.Fatal(err)
	}

	draft := Draft {
		Id: 36,
	}

	var owner *User
	for _, user := range users {
		if user.Username == "AgentOne" {
			owner = user
		}
	}

	// owner, draft := initDraft(users)
	slog.Info("Created draft", "Id", draft.Id)

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
			slog.Warn("failed to find picking player")
			time.Sleep(2 * time.Minute)
			continue
		}
		slog.Info("Picking Player", "PLayer", pickingPlayer.Username)

		slog.Info("Starting pick round", "Picking player", pickingPlayer.Username, "Continue", sameSession)

		nextPick, err := requestNextDraftPick(pickingPlayer, draft.Id, sameSession, additionalPrompt)
		if err != nil {
			slog.Info("Opencode pick was not a number", "Pick", nextPick, "Error", err)
			additionalPrompt = "Make sure your pick is only a single team number and contains no additional reasoning"
			sameSession = true
			continue
		}

		slog.Info("Got picking player", "Username", pickingPlayer.Username, "Pick", nextPick)

		pickMade, errMsg := makePickRequest(draft.Id, pickingPlayer, nextPick)
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

	slog.Info("Prompting opencode for a pick", "Model", pickingPlayer.Persona.Model, "Variant", pickingPlayer.Persona.Variant, "Prompt", prompt)

	resp, err := callOpencode(prompt, pickingPlayer.Persona.Model, pickingPlayer.Persona.Variant, flags...)
	if err != nil {
		return -1, err
	}
	slog.Info("Got next pick from opencode", "Pick Response", resp)


	return strconv.Atoi(strings.TrimSpace(resp))
}

func callOpencode(prompt string, model string, variant string, flags ...string) (string, error) {
	var args []string
	args = append(args, "run")
	if model != "" {
		args = append(args, "--model", model)
	}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
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
