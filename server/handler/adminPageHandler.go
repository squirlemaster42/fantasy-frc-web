package handler

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"

	"server/assert"
	"server/model"
	"server/view/admin"
)

type Command interface {
    ProcessCommand(inputs []string) string
    Help() string
}

type PingCommand struct { }

func (p *PingCommand) ProcessCommand(inputs []string) string {
    if len(inputs) > 0 {
        return "Ping does not take any inputs"
    }
    return "Pong"
}

func (p *PingCommand) Help() string {
    return "This command takes 0 argumets and returns pong"
}

var commands = map[string]Command {
    "ping": &PingCommand{},
}

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
    h.Logger.Log(fmt.Sprintf("Running command %s", splitCommandString[0]))

    if len(splitCommandString) < 1 {
        noCommandResponse := admin.RenderCommand(username, commandString, "")
        Render(c, noCommandResponse)
        return nil
    }

    command := commands[splitCommandString[0]]
    result := command.ProcessCommand(splitCommandString[1:])

    assert.AddContext("Command", commandString)

    response := admin.RenderCommand(username, commandString, result)
    Render(c, response)

    return nil
}
