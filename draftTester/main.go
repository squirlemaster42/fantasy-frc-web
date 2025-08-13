package main

import (
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
)

type User struct {
    Username string
    Password string
    AuthTok  string
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
}

func createDraft(user *User) Draft {
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

        cookies := resp.Cookies()
        for _, cookie := range cookies {
            if cookie.Name == "sessionToken" {
                slog.Info("Found auth token", "User", user.Username)
                user.AuthTok = cookie.Value
            }
        }
    }
}
