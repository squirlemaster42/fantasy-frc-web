package handler

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"server/assert"
	"server/logging"
	"server/model"
	"server/utils"
	"server/view/admin"
)

type Command interface {
    ProcessCommand(database *sql.DB, logger *logging.Logger, argStr string) string
}

type PingCommand struct { }

func (p *PingCommand) ProcessCommand(database *sql.DB, logger *logging.Logger, argStr string) string {
    if len(argStr) > 0 {
        return "Ping does not take any inputs"
    }
    return "Pong"
}

type ListDraftsCommand struct {}

func (l *ListDraftsCommand) ProcessCommand(database *sql.DB, logger *logging.Logger, argStr string) string {
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

type StartDraftCommand struct {}

func (s *StartDraftCommand) ProcessCommand(database *sql.DB, logger *logging.Logger, argStr string) string {
    argMap, _ := utils.ParseArgString(argStr)
    draftId, err := strconv.Atoi(argMap["id"])

    if err != nil {
        return "Draft Id Could Not Be Converted To An Int"
    }

    draft := model.GetDraft(database, draftId)

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

    //TODO This is not doing what I want it to
    model.MakePickAvailable(database, nextPickPlayer.Id, time.Now())

    // Need to start draft watch dog
    return "Draft Started"
}

var commands = map[string]Command {
    "ping": &PingCommand{},
    "listdraft": &ListDraftsCommand{},
    "startdraft": &StartDraftCommand{},
}

// ---------------- Handler Funcs --------------------------

func (h *Handler) HandleAdminConsoleGet(c echo.Context) error {
    h.Logger.Log("Got request to render admin console")
    assert := assert.CreateAssertWithContext("Handle Admin Console Get")
    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

    adminConsoleIndex := admin.AdminConsoleIndex(username)
    adminConsole := admin.AdminConsole(" | Admin Console", true, username, adminConsoleIndex)
    Render(c, adminConsole)

    return nil
}

func (h *Handler) HandleRunCommand(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Run Admin Console Command")
    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

	commandString := c.FormValue("command")
    cmd, args, _ := strings.Cut(commandString, " ")
    //This is to handle the case where we have no params
    h.Logger.Log(fmt.Sprintf("Running command %s with args %s", cmd, args))

    if len(cmd) < 1 {
        noCommandResponse := admin.RenderCommand(username, commandString, "")
        Render(c, noCommandResponse)
        return nil
    }

    command := commands[cmd]
    result := command.ProcessCommand(h.Database, h.Logger, args)

    assert.AddContext("Command", commandString)

    response := admin.RenderCommand(username, commandString, result)
    Render(c, response)

    return nil
}
