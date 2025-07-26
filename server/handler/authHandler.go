package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"server/handler/contracts"
	"server/model"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleLoginRequest(c echo.Context) error {
    slog.Info("Got request to login")
    body, err := io.ReadAll(c.Request().Body)

    if err != nil {
        slog.Warn("Failed to ready login request body", "Error", err)
        c.Response().Status = http.StatusBadRequest
        return errors.New("failed to read login request")
    }

    var loginRequest contracts.LoginRequest
    err = json.Unmarshal(body, &loginRequest)

    if err != nil {
        slog.Warn("Failed to parse json body", "Error", err, "Body", body)
        c.Response().Status = http.StatusBadRequest
        return errors.New("failed to parse login request")
    }

    slog.Info("Received login request", "Username", loginRequest.Username)

    //Validate the user and if they are valid signal that they should be
    //redirected to the draft list page. We also need to set the cookie header
    isValid := model.UsernameTaken(h.Database, loginRequest.Username)
    isValid = isValid && model.ValidateLogin(h.Database, loginRequest.Username, loginRequest.Password)

    if !isValid {
        c.Response().Status = http.StatusBadRequest
        return errors.New("invalid login attempt")
    }

    userUuid := model.GetUserUuidByUsername(h.Database, loginRequest.Username)
    sessionTok := generateSessionToken()
    model.RegisterSession(h.Database, userUuid, sessionTok)

    cookie := new(http.Cookie)
    cookie.Name = "sessionToken"
    cookie.Value = sessionTok
    cookie.HttpOnly = true
    cookie.Secure = true
    c.SetCookie(cookie)

    return nil
}
