package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"

	"draftclient"
	"golang.org/x/net/html"
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

type rawAgentUser struct {
	Username string         `json:"Username"`
	Password string         `json:"Password"`
	Persona  DrafterPersona `json:"DraftPersona"`
}

type AgentUser struct {
	User    *draftclient.User
	Persona DrafterPersona
}

func loadAgentUserConfig(path string) ([]*AgentUser, *draftclient.FantasyClient, error) {
	client := draftclient.New(target)

	userJSON, err := client.LoadUserConfig(path)
	if err != nil {
		return nil, nil, err
	}

	var rawUsers []rawAgentUser
	if err := json.Unmarshal([]byte(userJSON), &rawUsers); err != nil {
		return nil, nil, err
	}

	agentUsers := make([]*AgentUser, len(rawUsers))
	for i, ru := range rawUsers {
		agentUsers[i] = &AgentUser{
			User:    client.CreateUser(ru.Username, ru.Password),
			Persona: ru.Persona,
		}
	}

	return agentUsers, client, nil
}

type CurrentPicks struct {
	PlayerName string   `json:"playerName"`
	Picks      []string `json:"picks"`
}

func requestNextDraftPick(client *draftclient.FantasyClient, pickingPlayer *AgentUser, draftId int, sameSession bool, additionalPrompt string) (int, error) {
	var flags []string
	if sameSession {
		flags = append(flags, "-c")
	}

	currentPicks, err := getCurrentDraftPicks(client, pickingPlayer.User, draftId)
	if err != nil {
		return -1, nil
	}

	picksJSON, err := json.Marshal(currentPicks)
	if err != nil {
		return -1, err
	}

	prompt := fmt.Sprintf(
		"%s %s Your name is %s. The current picks in the draft for each player are %s. %s",
		SystemPrompt,
		pickingPlayer.Persona.PersonaPrompt,
		pickingPlayer.User.Username,
		picksJSON,
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

func getCurrentDraftPicks(client *draftclient.FantasyClient, user *draftclient.User, draftId int) ([]CurrentPicks, error) {
	resp, err := user.Client.Get(fmt.Sprintf("%s/u/draft/%d/pick", target, draftId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseCurrentDraftPicks(string(body)), nil
}

func parseCurrentDraftPicks(htmlContent string) []CurrentPicks {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}

	pickTable := draftclient.FindNodeById(doc, "pickTable")
	if pickTable == nil {
		return nil
	}

	var allPicks []CurrentPicks
	cards := draftclient.FindNodesByClass(pickTable, "border", "border-slate-600", "rounded-lg")
	for _, card := range cards {
		nameNode := draftclient.FindNodeByTagAndClass(card, "h3", "text-sm", "font-semibold", "text-slate-200", "text-center", "truncate")
		if nameNode == nil {
			continue
		}
		playerName := draftclient.ExtractText(nameNode)

		picksDiv := draftclient.FindNodeByClass(card, "p-3", "space-y-2", "bg-slate-800/50", "rounded-b-lg")
		if picksDiv == nil {
			continue
		}

		var picks []string
		for n := picksDiv.FirstChild; n != nil; n = n.NextSibling {
			if n.Type != html.ElementNode {
				continue
			}

			class := draftclient.GetAttribute(n, "class")
			if class == "" {
				continue
			}

			if draftclient.ContainsAll(class, "bg-blue-600") {
				picks = append(picks, draftclient.ExtractText(n))
			} else if draftclient.ContainsAll(class, "bg-red-600") {
				picks = append(picks, "Skipped")
			}
		}

		allPicks = append(allPicks, CurrentPicks{
			PlayerName: playerName,
			Picks:      picks,
		})
	}

	return allPicks
}
