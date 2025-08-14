package main

import (
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"reflect"
	"strings"
)

type User struct {
    Username string
    Password string
    Cookies []*http.Cookie
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
    users["UserOne"] = &User {
        Username: "UserOne",
        Password: "UserOne",
    }
    users["UserTwo"] = &User {
        Username: "UserTwo",
        Password: "UserTwo",
    }
    users["UserThree"] = &User {
        Username: "UserThree",
        Password: "UserThree",
    }
    users["UserFour"] = &User {
        Username: "UserFour",
        Password: "UserFour",
    }
    users["UserFive"] = &User {
        Username: "UserFive",
        Password: "UserFive",
    }
    users["UserSix"] = &User {
        Username: "UserSix",
        Password: "UserSix",
    }
    users["UserSeven"] = &User {
        Username: "UserSeven",
        Password: "UserSeven",
    }
    users["UserEight"] = &User {
        Username: "UserEight",
        Password: "UserEight",
    }

    populateAuthToks(users)

    //Choose a user and create a draft
    keys := reflect.ValueOf(users).MapKeys()
    createDraft(users[keys[rand.IntN(len(keys))].String()])
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

func createDraft(user *User) Draft {
    slog.Info("Making request to make draft", "User", user.Username)
    form := url.Values{}
    form.Add("description", createRandomString(10, 1000))
    form.Add("interval", "0")
    form.Add("startTime", "0001-01-01T00:00")
    form.Add("endTime", "0001-01-01T00:00")
    form.Add("draftName", createRandomString(5, 50))

    url, err := url.Parse(target)
    if err != nil {
        panic(err)
    }

    jar, err := cookiejar.New(nil)
    if err != nil {
        panic(err)
    }

    slog.Info("Adding cookies to request", "Cookies", jar)
    http.DefaultClient.Jar = jar
    http.DefaultClient.Jar.SetCookies(url, user.Cookies)
    resp, err := http.Post(target + "/u/createDraft", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
    if err != nil {
        slog.Error("Failed to create draft", "Username", user.Username, "Error", err)
        panic(err)
    }

    if resp.StatusCode != 200 {
        slog.Error("Failed to create draft", "User", user.Username)
        panic("failed to create draft")
    }

    body, err := io.ReadAll(resp.Body)
    slog.Info("Request made", "User", user.Username, "Status", resp.StatusCode, "Body", body)

    return Draft{}
}

func populateAuthToks(users map[string]*User) {
    for _, user := range users {
        slog.Info("Making login request", "User", user.Username)
        form := url.Values{}
        form.Add("username", user.Username)
        form.Add("password", user.Password)
        resp, err := http.Post(target + "/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
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

        user.Cookies = resp.Cookies()
    }
}
