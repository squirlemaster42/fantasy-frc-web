package main

import "testing"

func TestParseUserJSON(t *testing.T) {
	json := `[
    {
        "Username": "UserOne",
        "Password": "UserOne"
    },
    {
        "Username": "UserTwo",
        "Password": "UserTwo"
    },
    {
        "Username": "UserThree",
        "Password": "UserThree"
    },
    {
        "Username": "UserFour",
        "Password": "UserFour"
    },
    {
        "Username": "UserFive",
        "Password": "UserFive"
    },
    {
        "Username": "UserSix",
        "Password": "UserSix"
    },
    {
        "Username": "UserSeven",
        "Password": "UserSeven"
    },
    {
        "Username": "UserSeven",
        "Password": "UserSeven"
    }
	]`

	users, err := parseUsers(json)
	if err != nil {
		t.Fatal(err)
	}

	if len(users) != 8 {
		t.Fail()
	}

	if users[0].Username != "UserOne" {
		t.Fail()
	}

	if users[0].Password != "UserOne" {
		t.Fail()
	}
}
