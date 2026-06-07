package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"server/draft"
	"server/log"
	"server/model"
	"server/tbaHandler"
	"server/utils"
	"server/view/admin"
)

type Command interface {
	ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string
}

type PingCommand struct{}

func (p *PingCommand) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
	if len(argStr) > 0 {
		return "Ping does not take any inputs"
	}
	return "Pong"
}

type PopulateTeamsCommand struct{}

func (p *PopulateTeamsCommand) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
	if len(argStr) > 0 {
		return "PopulateTeams does not take any inputs"
	}

	log.Info(ctx, "Populating Teams")
	count := 0

	for _, event := range utils.Events() {
		log.Debug(ctx, "Creating teams for event", "Event", event)
		teams := tbaHandler.MakeTeamsAtEventRequest(ctx, event)
		for _, t := range teams {
			log.Debug(ctx, "Checking if team is needed", "Team", t.Key, "Event", event)
			team, err := teamStore.GetTeam(ctx, t.Key)
			if err != nil {
				log.Error(ctx, "Failed to get team", "Team", t.Key, "Error", err)
				continue
			}
			if team == nil {
				log.Debug(ctx, "Creating team", "Team", t.Key, "Event", event)
				if err := teamStore.CreateTeam(ctx, t.Key, ""); err != nil {
					log.Error(ctx, "Failed to create team", "Team", t.Key, "Error", err)
					continue
				}
				count++
			}
		}
	}

	log.Info(ctx, "Finished populating teams", "Count", count)
	return fmt.Sprintf("Successfully populated %d teams", count)
}

type ListDraftsCommand struct{}

func (l *ListDraftsCommand) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
	//Parse command inputs
	argMap, _ := utils.ParseArgString(argStr)
	searchString := argMap["s"]

	drafts, err := draftStore.GetDraftsByName(ctx, searchString)
	if err != nil {
		return "Failed to list drafts: " + err.Error()
	}

	var sb strings.Builder

	sb.WriteString("Id    |  Name\n")
	sb.WriteString("-------------\n")

	for _, draft := range drafts {
		sb.WriteString(fmt.Sprintf("%4d  | %s\n", draft.Id, draft.DisplayName))
	}

	return sb.String()
}

type StartDraftCommand struct{}

func (s *StartDraftCommand) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
	argMap, _ := utils.ParseArgString(argStr)
	draftId, err := strconv.Atoi(argMap["id"])

	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}


	draftActor, err := draftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return "Draft Id Does Not Match A Valid Draft"
	}
	draftState := draftActor.GetDraftState()

	//Check that eight players have accepted the draft
	numAccepted := 0
	for _, player := range draftState.Players {
		if !player.Pending {
			numAccepted += 1
		}
	}

	if numAccepted != 8 {
		return "Not Enough Players Have Accepted The Draft"
	}

	replyChan := make(chan draft.Result)
	message := draft.Message {
		Content: draft.StateTransitionMessage{
			RequestedState: model.WAITING_TO_START,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(ctx, message)
	if err != nil {
		log.Error(ctx, "Failed to post state transition message", "Draft Id", draftId, "Error", err)
	} else {
		select {
		case result := <- message.Reply:
			if result.Error != nil {
				err = result.Error
			}
		case <- time.After(5 * time.Second):
			log.Warn(ctx, "State transition timed out", "Draft Id", draftId, "Current Pick Id", draftActor.GetDraftState().CurrentPick.Id)
		}
	}

	if err != nil {
		log.Error(ctx, "Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		return err.Error()
	}

	// TODO Need to start draft watch dog
	return "Draft Started"
}

type ViewWebhookKey struct{}

func (s *ViewWebhookKey) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
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

func (s *SkipPickCommand) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
	log.Info(ctx, "Calling skip command", "Args", argStr)
	argMap, _ := utils.ParseArgString(argStr)
	draftId, err := strconv.Atoi(argMap["id"])

	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}

	draftActor, err := draftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return "Draft Id Does Not Match A Valid Draft"
	}

	replyChan := make(chan draft.Result)
	skipped := false
	message := draft.Message {
		Content: draft.SkipCurrentPickMessage {
			CurrentPickId: draftActor.GetDraftState().CurrentPick.Id,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(ctx, message)
	if err != nil {
		log.Error(ctx, "Failed to post skip message to draft actor", "Draft Id", draftId, "Error", err)
	} else {
		select {
		case result := <- message.Reply:
			if result.Error != nil {
				log.Warn(ctx, "Skipping current pick in draft failed", "Draft Id", draftId, "Current Pick Id", draftActor.GetDraftState().CurrentPick.Id, "Error", result.Error)
				err = result.Error
				skipped = false
			} else {
				skipped = true
			}
		case <- time.After(5 * time.Second):
			log.Warn(ctx, "Skipping current pick in draft timed out", "Draft Id", draftId, "Current Pick Id", draftActor.GetDraftState().CurrentPick.Id)
			skipped = false
		}
	}
	if err != nil {
		return "Failed to skip player: " + err.Error()
	}

	if skipped {
		return "Player was skipped"
	} else {
		return "Did not get confirmation of skip. Verify draft state"
	}
}

type ModifyPickTimeCommand struct{}

func (m *ModifyPickTimeCommand) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
	log.Info(ctx, "Calling modify pick time command", "Args", argStr)
	argMap, _ := utils.ParseArgString(argStr)

	draftIdStr, ok := argMap["id"]
	if !ok {
		return "Missing required argument: -id=<draftId>"
	}

	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}

	pickIdStr, ok := argMap["pickId"]
	if !ok {
		return "Missing required argument: -pickId=<pickid>"
	}

	pickId, err := strconv.Atoi(pickIdStr)
	if err != nil {
		return "Pick Id Could Not Be Converted To An Int"
	}

	durationStr, ok := argMap["time"]
	if !ok {
		return "Missing required argument: -time=<duration>"
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return "Invalid duration format. Use format like: 45m, 1h30m, 2h15m30s"
	}

	draftActor, err := draftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return "Draft Id Does Not Match A Valid Draft"
	}

	replyChan := make(chan draft.Result)
	message := draft.Message {
		Content: draft.ModifyExpirationTimeMessage{
			PickId: pickId,
			Extension: duration,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(ctx, message)
	if err != nil {
		log.Error(ctx, "Failed to post modify expiration time message", "Draft Id", draftId, "Error", err)
	} else {
		select {
		case result := <- message.Reply:
			if result.Error != nil {
				log.Warn(ctx, "Extending pick expiration time failed", "Draft Id", draftId, "Pick Id", pickId, "Error", result.Error)
				err = result.Error
			}
		case <- time.After(5 * time.Second):
			log.Warn(ctx, "Extending pick expiration time timed out", "Draft Id", draftId)
			err = errors.New("timeout extending pick expiration time")
		}
	}

	if err != nil {
		log.Warn(ctx, "Update draft pick expiration time failed", "Draft Id", draftId, "Duration", duration, "Error", err)
		return err.Error()
	}

	return "Successfully updated pick expiration time"
}

type AdminPickCommand struct{}

func (a *AdminPickCommand) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
	log.Info(ctx, "Calling admin pick command", "Args", argStr)
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

	draftActor, err := draftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return err.Error()
	}
	draftState := draftActor.GetDraftState()

	if draftState.CurrentPick.Id == 0 {
		return "No current pick found for this draft"
	}

	pickStruct := model.Pick{
		Id:     draftState.CurrentPick.Id,
		Player: draftState.CurrentPick.Player,
		Pick: sql.NullString{
			Valid:  true,
			String: tbaId,
		},
		PickTime: sql.NullTime{
			Valid: true,
			Time:  time.Now(),
		},
	}

	replyChan := make(chan draft.Result)
	message := draft.Message {
		Content: draft.PickMessage{
			Pick: pickStruct,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(ctx, message)
	if err != nil {
		log.Error(ctx, "Failed to post pick message", "Draft Id", draftId, "Error", err)
		return err.Error()
	}
	var pickError error
	select {
	case result := <- message.Reply:
		if result.Error != nil {
			pickError = result.Error
		}
	case <- time.After(5 * time.Second):
		log.Warn(ctx, "making pick in draft timed out", "Draft Id", draftId, "Current Pick Id", draftActor.GetDraftState().CurrentPick.Id)
		return "make pick timed out"
	}
	if pickError != nil {
		log.Warn(ctx, "Could Not Make Pick", "Current Pick", "Pick", tbaId, "Error", err)
		return err.Error()
	}

	return fmt.Sprintf("Successfully picked team %s", teamStr)
}

type RenameDraftCommand struct{}

func (r *RenameDraftCommand) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
	log.Info(ctx, "Calling rename draft command", "Args", argStr)
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
	draftModel, err := draftActorMap.GetDraft(ctx, draftId)
	if err != nil {
		return "Draft Id Does Not Match A Valid Draft"
	}

	oldName := draftModel.DisplayName
	draftModel.DisplayName = newName

	// Update the draft
	err = draftActorMap.UpdateDraft(ctx, draftModel)
	if err != nil {
		log.Error(ctx, "Failed to update draft name", "Draft Id", draftId, "Error", err)
		return "Failed to update draft name"
	}

	return fmt.Sprintf("Successfully renamed draft from '%s' to '%s'", oldName, newName)
}

type UndoPickCommand struct{}

func (u *UndoPickCommand) ProcessCommand(ctx context.Context, tbaHandler tbaHandler.TbaHandler, draftStore model.DraftStore, userStore model.UserStore, teamStore model.TeamStore, draftActorMap *draft.DraftActorMap, argStr string) string {
	log.Info(ctx, "Calling undo pick command", "Args", argStr)
	argMap, _ := utils.ParseArgString(argStr)

	draftIdStr, ok := argMap["id"]
	if !ok {
		return "Missing required argument: -id=<draftId>"
	}

	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		return "Draft Id Could Not Be Converted To An Int"
	}

	err = draftActorMap.UndoLastPick(ctx, draftId)
	if err != nil {
		return err.Error()
	}

	return "Successfully undid pick"
}

var commands = map[string]Command{
	"ping":           &PingCommand{},
	"populateTeams":  &PopulateTeamsCommand{},
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

	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		username = ""
	}

	adminConsoleIndex := admin.AdminConsoleIndex(username, h.csrfToken(c))
	adminConsole := admin.AdminConsole(" | Admin Console", true, username, adminConsoleIndex)
	return Render(c, adminConsole)
}

func (h *Handler) HandleRunCommand(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		username = ""
	}

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

	result := command.ProcessCommand(c.Request().Context(), h.TbaHandler, h.DraftStore, h.UserStore, h.TeamStore, h.DraftActorMap, args)

	response := admin.RenderCommand(username, commandString, result)
	return Render(c, response)
}
