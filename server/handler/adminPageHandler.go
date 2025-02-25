package handler

import (
	"flag"
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"

	"server/assert"
	"server/logging"
	"server/model"
	"server/view/admin"
)

type Command interface {
    ProcessCommand(logger *logging.Logger, command string) string
    Help() string
}

type PingCommand struct { }

func (p *PingCommand) ProcessCommand(logger *logging.Logger, command string) string {
    if len(command) > 4 {
        return "Ping does not take any inputs"
    }
    return "Pong"
}

func (p *PingCommand) Help() string {
    return "This command takes 0 argumets and returns pong"
}

type ListDraftsCommand struct {}

func (l *ListDraftsCommand) ProcessCommand(logger *logging.Logger, command string) string {
    //Parse command inputs
    fs := flag.NewFlagSet("ListDraftsCommand", flag.ContinueOnError)

    searchFlag := fs.String("s", "", "A search string to use to filter the drafts")
    args := strings.Split(command, " ")[1:]
    fs.Parse(args)

    logger.Log(fmt.Sprintf("Search String: %s", *searchFlag))

    return *searchFlag
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
    splitCommandString := strings.Split(commandString, " ")
    //This is to handle the case where we have no params
    splitCommandString = append(splitCommandString, "")
    h.Logger.Log(fmt.Sprintf("Running command %s", splitCommandString[0]))

    if len(splitCommandString) < 1 {
        noCommandResponse := admin.RenderCommand(username, commandString, "")
        Render(c, noCommandResponse)
        return nil
    }

    command := commands[splitCommandString[0]]
    result := command.ProcessCommand(h.Logger, commandString)

    assert.AddContext("Command", commandString)

    response := admin.RenderCommand(username, commandString, result)
    Render(c, response)

    return nil
}
