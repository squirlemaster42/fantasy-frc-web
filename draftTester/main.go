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
	"time"

	"github.com/google/uuid"
)

type User struct {
	Username string
	Password string
	Client   http.Client
	Uuid     string
	IsOwner  bool
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
	owner.IsOwner = true
	draft := createDraft(owner)
	invitePlayersToDraft(owner, users, draft)
	for _, user := range users {
		acceptInvite(user)
	}

	//Start Draft
	startDraft(owner, draft.Id)

	//Check that draft is in the correct status
	currentDraftStatus := getCurrentDraftStatus(owner, draft.Id)
	if getCurrentDraftStatus(owner, draft.Id) != "Waiting to Start" {
		slog.Error("Got unexpected draft status", "Expected", "Waiting to Start", "Actual", currentDraftStatus)
		panic("draft status is not correct")
	}

	// Wait for draft start time to hit and make sure draft goes into picking
	waitUntilDraftState(owner, draft.Id, "Picking", 100*time.Second)

	slog.Info("Starting to make picks")

	successfulPicks := 0
	invalidPicks := 0

	// Have play make picks in a random order. Some picks being valid and some being invalid
	for getCurrentDraftStatus(owner, draft.Id) != "Teams Playing" {
		var pickingPlayer *User
		for _, player := range users {
			if isPickingPlayer(player, draft.Id) {
				pickingPlayer = player
				break
			}
		}
		slog.Info("Got picking player", "Username", pickingPlayer.Username)
		if rand.IntN(10) < 3 {
			pickingPlayer = selectRandomPlayer(users)
			slog.Info("Chose random player instead", "Username", pickingPlayer.Username)
		}
		success := makePickRequest(draft.Id, pickingPlayer, getRandomTeamId())
		if success {
			successfulPicks++
		} else {
			invalidPicks++
		}

		if successfulPicks > 64 {
			panic("too many picks were made")
		}
		slog.Info("Picking round made", "Successful Picks", successfulPicks, "Invalid Picks", invalidPicks)
	}

	// Make sure the draft goes to teams playing status
}

func isPickingPlayer (user *User, draftId int) bool {
	slog.Info("Getting picking player", "Draft Id", draftId, "User", user.Username)

	resp, err := user.Client.Get(fmt.Sprintf("%s/u/draft/%d/pick", target, draftId))
	if err != nil {
		slog.Error("Failed to get pick page", "Draft Id", draftId, "Username", user.Username, "Error", err)
		panic(err)
	}

	if resp.StatusCode != 200 {
		slog.Error("Failed to get pick page", "Draft Id", draftId, "User", user.Username)
		panic("failed to get pick page")
	}

	slog.Info("Make pick request make", "Draft Id", draftId, "User", user.Username, "Status", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic("failed to read response body after attempting to get pick page")
	}
	defer resp.Body.Close()

	return strings.Contains(string(body), `name="pickInput"`)
}

func getRandomTeamId() int {
	return rand.IntN(10000)
}

// True if pick was made successfully
func makePickRequest(draftId int, user *User, team int) bool {
	slog.Info("Make Pick", "Draft Id", draftId, "User", user.Username, "Team", team)
	form := url.Values{}
	form.Add("pickInput", strconv.Itoa(team))

	resp, err := user.Client.Post(fmt.Sprintf("%s/u/draft/%d/makePick", target, draftId), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("Failed to make pick", "Team", team, "Draft Id", draftId, "Username", user.Username, "Error", err)
		panic(err)
	}

	if resp.StatusCode != 200 {
		slog.Error("Failed to make pick", "Team", team, "Draft Id", draftId, "User", user.Username)
		panic("failed to make pick")
	}

	slog.Info("Make pick request make", "Draft Id", draftId, "User", user.Username, "Status", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic("failed to read response body after attempting to make pick")
	}
	defer resp.Body.Close()

	return !strings.Contains(string(body), "id=\"pickError\"")
}

func selectRandomPlayer(users map[string]*User) *User {
	keys := reflect.ValueOf(users).MapKeys()
	return users[keys[rand.IntN(len(keys))].String()]
}

// This will block until the draft is in the desired state or the timeout is hit. Timeout is in milliseconds
func waitUntilDraftState(user *User, draftId int, requestedStatus string, timeout time.Duration) {
	waitTime := 30 * time.Second
	timeoutTime := time.Now().Add(timeout)
	currentStatus := getCurrentDraftStatus(user, draftId)
	for currentStatus != requestedStatus {
		slog.Info("Checking if current draft is in requested status", "Requested Status", requestedStatus, "Current Status", currentStatus)
		if time.Now().After(timeoutTime) {
			panic("wait until draft state timeout reached")
		}

		currentStatus = getCurrentDraftStatus(user, draftId)
		time.Sleep(waitTime)
	}
}

func startDraft(user *User, draftId int) {
	slog.Info("Start Draft", "Draft Id", draftId, "User", user.Username)
	form := url.Values{}

	resp, err := user.Client.Post(fmt.Sprintf("%s/u/draft/%d/startDraft", target, draftId), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("Failed to start draft", "Draft Id", draftId, "Username", user.Username, "Error", err)
		panic(err)
	}

	if resp.StatusCode != 200 {
		slog.Error("Failed to start draft", "Draft Id", draftId, "User", user.Username)
		panic("failed to create draft")
	}

	//body, err := io.ReadAll(resp.Body)
	slog.Info("Start Draft Request Made", "Draft Id", draftId, "User", user.Username, "Status", resp.StatusCode)

	slog.Info("Started Draft", "Draft Id", draftId)
}

func acceptInvite(user *User) {
	//Find the draft id, accept the invite, repeat until we dont find any more draft ids
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/u/viewInvites", target), nil)
	if err != nil {
		panic(err)
	}

	resp, err := user.Client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var acceptRespBody string
	id, found := getInviteId(string(body))
	r := 0
	for {
		if found {
			slog.Info("Sending accept invite request", "User", user.Username, "Id", id)
			acceptRespBody = sendAcceptInvite(user, id)
			break
		} else if r > 5 {
			if user.IsOwner {
				break
			}
			slog.Error("Did not find at least one invite id", "Username", user.Username)
			panic("error: did not find at least one invite id")
		} else {
			r++
			time.Sleep(500 * time.Millisecond)
			id, found = getInviteId(string(body))
		}
	}

	for found {
		id, found = getInviteId(acceptRespBody)
		if found {
			acceptRespBody = sendAcceptInvite(user, id)
		}
	}
}

func getDraftIdPage(user *User, draftId int) string {
	//Find the draft id, accept the invite, repeat until we dont find any more draft ids
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/u/draft/%d/profile", target, draftId), nil)
	if err != nil {
		panic(err)
	}

	resp, err := user.Client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(body)
}

func getCurrentDraftStatus(user *User, draftId int) string {
	//Get Draft Profile package
	profilePage := getDraftIdPage(user, draftId)

	//Parse out the status
	return parseDraftStatus(profilePage)
}

func parseDraftStatus(profilePage string) string {
	prefix := "id=\"draftStatus\">"
	idx := strings.Index(profilePage, prefix)
	if idx == -1 {
		return ""
	}
	idx += len(prefix)
	endIdx := strings.Index(profilePage[idx:], "</div>")
	if endIdx == -1 {
		return ""
	}
	status := strings.TrimSpace(profilePage[idx : idx+endIdx])
	return status
}

func sendAcceptInvite(user *User, inviteId int) string {
	//This should return the respose of the accept request
	form := url.Values{}
	form.Add("inviteId", strconv.Itoa(inviteId))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/u/acceptInvite", target), strings.NewReader(form.Encode()))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := user.Client.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != 200 {
		panic("failed to accept invite")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(body)
}

func getInviteId(body string) (int, bool) {
	prefix := "<button hx-target=\"#pendingTable\" hx-swap=\"outerHTML\" name=\"inviteId\" value=\""
	if strings.Count(body, prefix) == 0 {
		return -1, false
	}

	idx := strings.Index(string(body), prefix) + len(prefix)
	sliced := string(body)[idx:]
	endIdx := strings.Index(sliced, "\"")
	id, err := strconv.Atoi(sliced[:endIdx])

	if err != nil {
		panic(err)
	}

	return id, true
}

func createUser(username string) *User {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &User{
		Username: username,
		Password: username,
		Client: http.Client{
			Jar: jar,
		},
	}
}

func createRandomString(minLen int, maxLen int) string {
	alphabet := "abcdefghijklmnopqrstuvwxyz"
	length := rand.IntN(maxLen-minLen) + minLen
	var sb strings.Builder
	for range length {
		sb.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return sb.String()
}

func getPlayerUUID(owner *User, draftId int, username string) uuid.UUID {
	form := url.Values{}
	form.Add("description", "")
	form.Add("interval", "0")
	form.Add("startTime", "0001-01-01T00:00")
	form.Add("endTime", "0001-01-01T00:00")
	form.Add("draftName", "")
	form.Add("search", username)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/u/searchPlayers", target), strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("Failed to create new request", "Error", err)
		panic(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Hx-Current-Url", fmt.Sprintf("%s/u/draft/%d/profile", target, draftId))

	resp, err := owner.Client.Do(req)
	if err != nil {
		slog.Error("Failed to search for player", "Username", username, "Error", err)
		panic(err)
	}

	if resp.StatusCode != 200 {
		slog.Error("Failed to search for username", "User", username)
		panic("failed to create draft")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	slog.Info("Request made", "User", username, "Status", resp.StatusCode)

	prefix := "<button hx-target=\"#inviteTable\" hx-swap=\"outerHTML\" name=\"userUuid\" value=\""
	if strings.Count(string(body), prefix) != 1 {
		slog.Error("Did not find only one user", "Username", username, "Draft Id", draftId)
		panic("err: did not find only one user")
	}

	idx := strings.Index(string(body), prefix) + len(prefix)
	sliced := string(body)[idx:]
	uuidStr := sliced[:strings.Index(sliced, "\"")]

	uuid, err := uuid.Parse(uuidStr)

	if err != nil {
		panic(err)
	} else {
		slog.Info("Found UUID", "Username", username, "UUID", uuid)
	}

	return uuid
}

func invitePlayersToDraft(owner *User, users map[string]*User, draft Draft) {
	for _, user := range users {
		if user.Username == owner.Username {
			continue
		}

		form := url.Values{}
		form.Add("description", createRandomString(10, 1000))
		form.Add("interval", "0")
		form.Add("startTime", "0001-01-01T00:00")
		form.Add("endTime", "0001-01-01T00:00")
		form.Add("draftName", createRandomString(5, 50))
		form.Add("search", "")
		form.Add("userUuid", getPlayerUUID(owner, draft.Id, user.Username).String())

		resp, err := owner.Client.Post(fmt.Sprintf("%s/u/draft/%d/invitePlayer", target, draft.Id), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		if err != nil {
			slog.Error("Failed to invite player to draft", "Username", user.Username, "Error", err)
			panic(err)
		}

		if resp.StatusCode != 200 {
			slog.Error("Failed to invite player to draft", "User", user.Username)
			panic("failed to create draft")
		}

		//body, err := io.ReadAll(resp.Body)
		slog.Info("Request made", "User", user.Username, "Status", resp.StatusCode)
	}
}

func createDraft(user *User) Draft {
	slog.Info("Making request to make draft", "User", user.Username)

	startTime := time.Now().Add(1 * time.Minute)

	form := url.Values{}
	form.Add("description", createRandomString(10, 1000))
	form.Add("interval", "0")
	form.Add("startTime", startTime.Format(time.RFC3339))
	form.Add("endTime", startTime.Add(10*time.Minute).Format(time.RFC3339))
	form.Add("draftName", createRandomString(5, 50))

	resp, err := user.Client.Post(target+"/u/createDraft", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("Failed to create draft", "Username", user.Username, "Error", err)
		panic(err)
	}

	if resp.StatusCode != 200 {
		slog.Error("Failed to create draft", "User", user.Username)
		panic("failed to create draft")
	}

	//body, err := io.ReadAll(resp.Body)
	slog.Info("Create Draft Request Made", "User", user.Username, "Status", resp.StatusCode)

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
		resp, err := user.Client.Post(target+"/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		if err != nil {
			slog.Error("Failed login", "Username", user.Username, "Error", err)
			panic(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			slog.Error("Failed to login", "User", user.Username)
			panic("failed to login")
		}

		//body, err := io.ReadAll(resp.Body)
		slog.Info("Populate auth token request made", "User", user.Username, "Status", resp.StatusCode)
	}
}
