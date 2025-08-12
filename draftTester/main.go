package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type User struct {
    Username string
    Password string
    AuthTok  string
}

const (
    target = "http://localhost"
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
}

func populateAuthToks(users map[string]*User) {
    for _, user := range users {
        form := url.Values{}
        form.Add("username", user.Username)
        form.Add("password", user.Password)
        resp, err := http.Post(target + "/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
        if err != nil {
            slog.Error("Failed login", "Username", user.Username, "Error", err)
            panic(err)
        }
        defer resp.Body.Close()
        cookies := resp.Cookies()
        for _, cookie := range cookies {
            if cookie.Name == "sessionToken" {
                user.AuthTok = cookie.Value
            }
        }
    }
}
