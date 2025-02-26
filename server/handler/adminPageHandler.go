package handler

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"

	"server/assert"
	"server/logging"
	"server/model"
	"server/utils"
	"server/view/admin"
)

type Command interface {
    ProcessCommand(database *sql.DB, logger *logging.Logger, argStr string) string
    Help() string
}

type PingCommand struct { }

func (p *PingCommand) ProcessCommand(database *sql.DB, logger *logging.Logger, argStr string) string {
    if len(argStr) > 0 {
        return "Ping does not take any inputs"
    }
    return "Pong"
}

func (p *PingCommand) Help() string {
    return "This command takes 0 argumets and returns pong"
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

func (l *ListDraftsCommand) Help() string {
    return "This command lists the drafts. -s \"<name>\" allows filtering drafts by name."
}

var commands = map[string]Command {
    "ping": &PingCommand{},
    "listdraft": &ListDraftsCommand{},
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
