package main

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type User struct {
    Username string
    Password string
    Client http.Client
    Uuid string
}

type Draft struct {
    Id int
}

const (
    target = "http://localhost:7331"
)

func main() {

    // Map Username to user struct
    users := make(map[string]*User)

    //Build map of usernames and passwords
    users["UserOne"] = createUser("UserOne")
    users["UserTwo"] = createUser("UserTwo")
    users["UserThree"] = createUser("UserThree")
    users["UserFour"] = createUser("UserFour")
    users["UserFive"] = createUser("UserFive")
    users["UserSix"] = createUser("UserSix")
    users["UserSeven"] = createUser("UserSeven")
    users["UserEight"] = createUser("UserEight")
    populateAuthToks(users)

    //Choose a user and create a draft
    keys := reflect.ValueOf(users).MapKeys()
    owner := users[keys[rand.IntN(len(keys))].String()]
    draft := createDraft(owner)
    invitePlayersToDraft(owner, users, draft)
}

func createUser (username string) *User {
    jar, err := cookiejar.New(nil)
    if err != nil {
        panic(err)
    }
    return &User {
        Username: username,
        Password: username,
        Client: http.Client{
            Jar: jar,
        },
    }
}

func createRandomString(minLen int, maxLen int) string {
    alphabet := "abcdefghijklmnopqrstuvwxyz"
    length := rand.IntN(maxLen - minLen) + minLen
    var sb strings.Builder
    for range length {
        sb.WriteByte(alphabet[rand.IntN(len(alphabet))])
    }
    return sb.String()
}

func invitePlayersToDraft(owner *User, users map[string]*User, draft Draft) {
    for _, user := range users {
        form := url.Values{}
        form.Add("description", createRandomString(10, 1000))
        form.Add("interval", "0")
        form.Add("startTime", "0001-01-01T00:00")
        form.Add("endTime", "0001-01-01T00:00")
        form.Add("draftName", createRandomString(5, 50))
        form.Add("search", "")

        //TODO Need to figure out how to get uuid
        form.Add("userUuid", user.Uuid)

        resp, err := owner.Client.Post(fmt.Sprintf("%s/u/draft/%d/invitePlayer", target, draft.Id), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
        if err != nil {
            slog.Error("Failed to invite player to draft", "Username", user.Username, "Error", err)
            panic(err)
        }

        if resp.StatusCode != 200 {
            slog.Error("Failed to invite player to draft", "User", user.Username)
            panic("failed to create draft")
        }

        body, err := io.ReadAll(resp.Body)
        slog.Info("Request made", "User", user.Username, "Status", resp.StatusCode, "Body", body, "Headers", resp.Header)
    }
}

func createDraft(user *User) Draft {
    slog.Info("Making request to make draft", "User", user.Username)
    form := url.Values{}
    form.Add("description", createRandomString(10, 1000))
    form.Add("interval", "0")
    form.Add("startTime", "0001-01-01T00:00")
    form.Add("endTime", "0001-01-01T00:00")
    form.Add("draftName", createRandomString(5, 50))

    resp, err := user.Client.Post(target + "/u/createDraft", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
    if err != nil {
        slog.Error("Failed to create draft", "Username", user.Username, "Error", err)
        panic(err)
    }

    if resp.StatusCode != 200 {
        slog.Error("Failed to create draft", "User", user.Username)
        panic("failed to create draft")
    }

    body, err := io.ReadAll(resp.Body)
    slog.Info("Request made", "User", user.Username, "Status", resp.StatusCode, "Body", body, "Headers", resp.Header)

    // Get Draft Id
    draftIdStr := strings.Split(resp.Header.Get("Hx-Redirect"), "/")[3]
    slog.Info("Parsed draft id string", "Draft Id String", draftIdStr)
    draftId, err := strconv.Atoi(draftIdStr)
    if err != nil {
        slog.Error("Failed to parse draft id from redirect")
        panic(err)
    }

    slog.Info("Created Draft", "Id", draftId)

    return Draft{
        Id: draftId,
    }
}

func populateAuthToks(users map[string]*User) {
    for _, user := range users {
        slog.Info("Making login request", "User", user.Username)
        form := url.Values{}
        form.Add("username", user.Username)
        form.Add("password", user.Password)
        resp, err := user.Client.Post(target + "/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
        if err != nil {
            slog.Error("Failed login", "Username", user.Username, "Error", err)
            panic(err)
        }

        defer resp.Body.Close()

        if resp.StatusCode != 200 {
            slog.Error("Failed to login", "User", user.Username)
            panic("failed to login")
        }

        body, err := io.ReadAll(resp.Body)
        slog.Info("Request made", "User", user.Username, "Status", resp.StatusCode, "Body", body)
    }
}
