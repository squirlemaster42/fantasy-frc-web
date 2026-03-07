package handler

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"server/assert"
	"server/draft"
	"server/log"
	"server/model"
	"server/utils"
	"server/view/admin"
)

type Command interface {
	ProcessCommand(database *sql.DB, draftManager *draft.DraftManager, argStr string) string
}

type PingCommand struct{}

func (p *PingCommand) ProcessCommand(database *sql.DB, draftManager *draft.DraftManager, argStr string) string {
	if len(argStr) > 0 {
		return "Ping does not take any inputs"
	}
	return "Pong"
}

type ListDraftsCommand struct{}

func (l *ListDraftsCommand) ProcessCommand(database *sql.DB, draftManager *draft.DraftManager, argStr string) string {
	//Parse command inputs
	argMap, _ := utils.ParseArgString(argStr)
	searchString := argMap["s"]

	drafts := model.GetDraftsByName(database, searchString)

	var sb strings.Builder

	sb.WriteString("Id    |  Name\n")
	sb.WriteString("-------------\n")

	for _, draft := range *drafts {
		sb.WriteString(fmt.Sprintf("%4d  | %s\n", draft.Id, draft.DisplayName))
	}

	return sb.String()
}

type StartDraftCommand struct{}

func (s *StartDraftCommand) ProcessCommand(database *sql.DB, draftManager *draft.DraftManager, argStr string) string {
	argMap, _ := utils.ParseArgString(argStr)
	draftId, err := strconv.Atoi(argMap["id"])

	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}

	draft, err := model.GetDraft(database, draftId)
	if err != nil {
		return "Draft Id Does Not Match A Valid Draft"
	}

	//Check that eight players have accepted the draft
	numAccepted := 0
	for _, player := range draft.Players {
		if !player.Pending {
			numAccepted += 1
		}
	}

	if numAccepted != 8 {
		return "Not Enough Players Have Accepted The Draft"
	}

	//Randomize pick order
	model.RandomizePickOrder(database, draftId)

	model.StartDraft(database, draftId)

	//Get the next pick and ready up that pick
	nextPickPlayer := model.NextPick(database, draftId)

	model.MakePickAvailable(database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))

	// Need to start draft watch dog
	return "Draft Started"
}

type ViewWebhookKey struct{}

func (s *ViewWebhookKey) ProcessCommand(database *sql.DB, draft *draft.DraftManager, argStr string) string {
	file, err := os.Open(utils.GetWebhookFilePath())
	if err != nil {
		return "Failed to open file: " + err.Error()
	}
	body, err := io.ReadAll(file)
	if err != nil {
		return "Failed to read file: " + err.Error()
	}

	return string(body)
}

type SkipPickCommand struct{}

func (s *SkipPickCommand) ProcessCommand(database *sql.DB, draftManager *draft.DraftManager, argStr string) string {
	log.InfoNoContext("Calling skip command", "Args", argStr)
	argMap, _ := utils.ParseArgString(argStr)
	draftId, err := strconv.Atoi(argMap["id"])

	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}

	err = draftManager.SkipCurrentPick(draftId)
	if err != nil {
		return "Failed to skip player: " + err.Error()
	}

	return "Player was skipped"
}

type ModifyPickTimeCommand struct{}

func (m *ModifyPickTimeCommand) ProcessCommand(database *sql.DB, draftManager *draft.DraftManager, argStr string) string {
	log.InfoNoContext("Calling modify pick time command", "Args", argStr)
	argMap, _ := utils.ParseArgString(argStr)

	draftIdStr, ok := argMap["id"]
	if !ok {
		return "Missing required argument: -id=<draftId>"
	}

	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}

	durationStr, ok := argMap["time"]
	if !ok {
		return "Missing required argument: -time=<duration>"
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return "Invalid duration format. Use format like: 45m, 1h30m, 2h15m30s"
	}

	err = draftManager.ModifyCurrentPickExpirationTime(draftId, duration)
	if err != nil {
		log.WarnNoContext("Update draft pick expiration time failed", "Draft Id", draftId, "Duration", duration, "Error", err)
		return err.Error()
	}

	return "Successfully updated pick expiration time"
}

type AdminPickCommand struct{}

func (a *AdminPickCommand) ProcessCommand(database *sql.DB, draftManager *draft.DraftManager, argStr string) string {
	log.InfoNoContext("Calling admin pick command", "Args", argStr)
	argMap, _ := utils.ParseArgString(argStr)

	draftIdStr, ok := argMap["id"]
	if !ok {
		return "Missing required argument: -id=<draftId>"
	}

	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}

	teamStr, ok := argMap["team"]
	if !ok {
		return "Missing required argument: -team=<teamNumber>"
	}

	// Format team ID (e.g., "254" -> "frc254")
	tbaId := "frc" + teamStr

	// Get the current pick
	currentPick, err := draftManager.GetCurrentPick(draftId)
	if currentPick.Id == 0 || err != nil {
		return "No current pick found for this draft"
	}

	// Build the pick struct
	pick := model.Pick{
		Id:       currentPick.Id,
		Player:   currentPick.Player,
		Pick:     sql.NullString{String: tbaId, Valid: true},
		PickTime: sql.NullTime{Time: time.Now(), Valid: true},
	}

	// Make the pick (this handles all validation)
	err = draftManager.MakePick(draftId, pick)
	if err != nil {
		return "Failed to make pick: " + err.Error()
	}

	return fmt.Sprintf("Successfully picked team %s", teamStr)
}

type RenameDraftCommand struct{}

func (r *RenameDraftCommand) ProcessCommand(database *sql.DB, draftManager *draft.DraftManager, argStr string) string {
	log.InfoNoContext("Calling rename draft command", "Args", argStr)
	argMap, _ := utils.ParseArgString(argStr)

	draftIdStr, ok := argMap["id"]
	if !ok {
		return "Missing required argument: -id=<draftId>"
	}

	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}

	newName, ok := argMap["name"]
	if !ok {
		return "Missing required argument: -name=<newName>"
	}

	if newName == "" {
		return "Draft name cannot be empty"
	}

	// Fetch the draft
	draft, err := draftManager.GetDraft(draftId, true)
	if err != nil {
		return "Draft Id Does Not Match A Valid Draft"
	}

	oldName := draft.Model.DisplayName
	draft.Model.DisplayName = newName

	// Update the draft
	err = draftManager.UpdateDraft(*draft.Model)
	if err != nil {
		log.ErrorNoContext("Failed to update draft name", "Draft Id", draftId, "Error", err)
		return "Failed to update draft name"
	}

	return fmt.Sprintf("Successfully renamed draft from '%s' to '%s'", oldName, newName)
}

type UndoPickCommand struct{}

func (u *UndoPickCommand) ProcessCommand(database *sql.DB, draftManager *draft.DraftManager, argStr string) string {
	log.InfoNoContext("Calling undo pick command", "Args", argStr)
	argMap, _ := utils.ParseArgString(argStr)

	draftIdStr, ok := argMap["id"]
	if !ok {
		return "Missing required argument: -id=<draftId>"
	}

	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}

	err = draftManager.UndoLastPick(draftId)
	if err != nil {
		return err.Error()
	}

	return "Successfully undid pick. Player %s now has until %s to make their pick"
}

var commands = map[string]Command{
	"ping":           &PingCommand{},
	"listdraft":      &ListDraftsCommand{},
	"startdraft":     &StartDraftCommand{},
	"skippick":       &SkipPickCommand{},
	"viewWebhookKey": &ViewWebhookKey{},
	"modifypicktime": &ModifyPickTimeCommand{},
	"adminpick":      &AdminPickCommand{},
	"renamedraft":    &RenameDraftCommand{},
	"undopick":       &UndoPickCommand{},
}

// ---------------- Handler Funcs --------------------------

func (h *Handler) HandleAdminConsoleGet(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to render admin console")
	assert := assert.CreateAssertWithContext("Handle Admin Console Get")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)

	adminConsoleIndex := admin.AdminConsoleIndex(username)
	adminConsole := admin.AdminConsole(" | Admin Console", true, username, adminConsoleIndex)
	return Render(c, adminConsole)
}

func (h *Handler) HandleRunCommand(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Run Admin Console Command")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)

	commandString := c.FormValue("command")
	cmd, args, _ := strings.Cut(commandString, " ")
	//This is to handle the case where we have no params
	log.Info(c.Request().Context(), "Running command", "Command", cmd, "Argument", args)

	if len(cmd) < 1 {
		noCommandResponse := admin.RenderCommand(username, commandString, "")
		return Render(c, noCommandResponse)
	}

	command, ok := commands[cmd]
	if !ok {
		response := admin.RenderCommand(username, commandString, "Invalid command")
		return Render(c, response)
	}
	result := command.ProcessCommand(h.Database, h.DraftManager, args)

	assert.AddContext("Command", commandString)

	response := admin.RenderCommand(username, commandString, result)
	return Render(c, response)
}
