package draftclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"uuid"

	"golang.org/x/net/html"
)

// User represents a Fantasy FRC user account that can make HTTP requests
type User struct {
	Username string
	Password string
	Client   http.Client
	Uuid     string
	IsOwner  bool
}

// Draft represents a Fantasy FRC draft
type Draft struct {
	Id int
}

// FantasyClient provides HTTP helpers for interacting with a Fantasy FRC server.
type FantasyClient struct {
	BaseURL string
}

// New creates a FantasyClient for the given base URL.
func New(baseURL string) *FantasyClient {
	return &FantasyClient{BaseURL: baseURL}
}

// LoadUserConfig reads the raw JSON at the given path.
func (fc *FantasyClient) LoadUserConfig(path string) (string, error) {
	file, err := os.ReadFile(path)
	return string(file), err
}

// ParseUsers parses a JSON array of users.
func (fc *FantasyClient) ParseUsers(userJSON string) ([]*User, error) {
	var users []*User
	err := json.Unmarshal([]byte(userJSON), &users)
	return users, err
}

// CreateUser creates a new User with an empty cookie jar.
func (fc *FantasyClient) CreateUser(username, password string) *User {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &User{
		Username: username,
		Password: password,
		Client: http.Client{
			Jar: jar,
		},
	}
}

// Login authenticates the user, verifies a session cookie was set, and primes
// the authenticated CSRF token by visiting a protected route.
func (fc *FantasyClient) Login(user *User) {
	slog.Info("Making login request", "User", user.Username)

	// 1. GET the login page to receive the csrf_cookie.
	resp, err := user.Client.Get(fc.BaseURL + "/login")
	if err != nil {
		slog.Error("Failed to get login page", "Username", user.Username, "Error", err)
		panic(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// 2. Extract the csrf_cookie value from the jar.
	csrfToken, ok := fc.getCookieValue(&user.Client, fc.BaseURL+"/login", "csrf_cookie")
	if !ok {
		panic("no csrf_cookie received from login page")
	}

	// 3. POST login with username, password, and csrf_token.
	form := url.Values{}
	form.Add("username", user.Username)
	form.Add("password", user.Password)
	form.Add("csrf_token", csrfToken)

	resp, err = user.Client.Post(fc.BaseURL+"/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("Failed login", "Username", user.Username, "Error", err)
		panic(err)
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		slog.Error("Failed to login", "User", user.Username, "Body", body, "Error", err)
		panic("failed to login")
	}

	// 4. Verify we actually got a session.
	_, hasSession := fc.getCookieValue(&user.Client, fc.BaseURL+"/login", "sessionToken")
	if !hasSession {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		slog.Error("Login succeeded but no sessionToken cookie was set", "Body", string(body))
		panic("login succeeded but no sessionToken cookie was set")
	}
	resp.Body.Close()

	// 5. Prime the authenticated CSRF token by making a GET to a protected route.
	resp, err = user.Client.Get(fc.BaseURL + "/u/home")
	if err != nil {
		slog.Error("Failed to get home page", "Username", user.Username, "Error", err)
		panic(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	slog.Info("Login request made", "User", user.Username, "Status", resp.StatusCode)
}

// CreateDraft creates a new draft and returns it.
func (fc *FantasyClient) CreateDraft(user *User) Draft {
	slog.Info("Making request to make draft", "User", user.Username)

	csrfToken, ok := fc.getCookieValue(&user.Client, fc.BaseURL+"/u/home", "csrf_token")
	if !ok {
		panic("no csrf_token cookie available for authenticated request")
	}

	startTime := time.Now().UTC().Add(1 * time.Minute)
	layout := "2006-01-02T15:04:05"
	form := url.Values{}
	form.Add("description", fc.CreateRandomString(10, 1000))
	form.Add("interval", "0")
	form.Add("startTime", startTime.Format(layout))
	form.Add("endTime", startTime.Add(10*time.Minute).Format(layout))
	form.Add("draftName", fc.CreateRandomString(5, 50))
	form.Add("csrf_token", csrfToken)

	resp, err := user.Client.Post(fc.BaseURL+"/u/createDraft", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("Failed to create draft", "Username", user.Username, "Error", err)
		panic(err)
	}

	if resp.StatusCode != 200 {
		slog.Error("Failed to create draft", "User", user.Username)
		panic("failed to create draft")
	}

	slog.Info("Create Draft Request Made", "User", user.Username, "Status", resp.StatusCode)

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

// GetPlayerUUID searches for a player by username and returns their UUID.
func (fc *FantasyClient) GetPlayerUUID(owner *User, draftId int, username string) uuid.UUID {
	csrfToken, ok := fc.getCookieValue(&owner.Client, fc.BaseURL+"/u/home", "csrf_token")
	if !ok {
		panic("no csrf_token cookie available for authenticated request")
	}

	form := url.Values{}
	form.Add("description", "")
	form.Add("interval", "0")
	form.Add("startTime", "0001-01-01T00:00")
	form.Add("endTime", "0001-01-01T00:00")
	form.Add("draftName", "")
	form.Add("search", username)
	form.Add("csrf_token", csrfToken)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/u/searchPlayers", fc.BaseURL), strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("Failed to create new request", "Error", err)
		panic(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Hx-Current-Url", fmt.Sprintf("%s/u/draft/%d/profile", fc.BaseURL, draftId))

	resp, err := owner.Client.Do(req)
	if err != nil {
		slog.Error("Failed to search for player", "Username", username, "Error", err)
		panic(err)
	}

	if resp.StatusCode != 200 {
		slog.Error("Failed to search for username", "User", username)
		panic("failed to search for username")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	slog.Info("Request made", "User", username, "Status", resp.StatusCode)

	uuidStr, found := fc.extractUserUuid(string(body))
	if !found {
		slog.Error("Did not find only one user", "Username", username, "Draft Id", draftId)
		panic("err: did not find only one user")
	}

	parsedUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		panic(err)
	}

	slog.Info("Found UUID", "Username", username, "UUID", parsedUuid)
	return parsedUuid
}

// extractUserUuid parses HTML and returns the value of the first element with
// name="userUuid".
func (fc *FantasyClient) extractUserUuid(htmlContent string) (string, bool) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", false
	}

	var values []string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "name" && attr.Val == "userUuid" {
					for _, a := range n.Attr {
						if a.Key == "value" {
							values = append(values, a.Val)
							break
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

// InvitePlayersToDraft invites all users except the owner to the draft.
func (fc *FantasyClient) InvitePlayersToDraft(owner *User, users []*User, draft Draft) {
	for _, user := range users {
		if user.Username == owner.Username {
			continue
		}

		csrfToken, ok := fc.getCookieValue(&owner.Client, fc.BaseURL+"/u/home", "csrf_token")
		if !ok {
			panic("no csrf_token cookie available for authenticated request")
		}

		form := url.Values{}
		form.Add("description", fc.CreateRandomString(10, 1000))
		form.Add("interval", "0")
		form.Add("startTime", "0001-01-01T00:00")
		form.Add("endTime", "0001-01-01T00:00")
		form.Add("draftName", fc.CreateRandomString(5, 50))
		form.Add("search", "")
		form.Add("userUuid", fc.GetPlayerUUID(owner, draft.Id, user.Username).String())
		form.Add("csrf_token", csrfToken)

		resp, err := owner.Client.Post(fmt.Sprintf("%s/u/draft/%d/invitePlayer", fc.BaseURL, draft.Id), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		if err != nil {
			slog.Error("Failed to invite player to draft", "Username", user.Username, "Error", err)
			panic(err)
		}

		if resp.StatusCode != 200 {
			slog.Error("Failed to invite player to draft", "User", user.Username)
			panic("failed to invite player")
		}

		slog.Info("Request made", "User", user.Username, "Status", resp.StatusCode)
	}
}

// AcceptInvite accepts the first pending draft invite for the user.
func (fc *FantasyClient) AcceptInvite(user *User) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/u/viewInvites", fc.BaseURL), nil)
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
	id, found := fc.getInviteId(string(body))
	r := 0
	for {
		if found {
			slog.Info("Sending accept invite request", "User", user.Username, "Id", id)
			acceptRespBody = fc.sendAcceptInvite(user, id)
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
			resp, err = user.Client.Get(fmt.Sprintf("%s/u/viewInvites", fc.BaseURL))
			if err != nil {
				panic(err)
			}
			body, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				panic(err)
			}
			id, found = fc.getInviteId(string(body))
		}
	}

	for found {
		id, found = fc.getInviteId(acceptRespBody)
		if found {
			acceptRespBody = fc.sendAcceptInvite(user, id)
		}
	}
}

func (fc *FantasyClient) sendAcceptInvite(user *User, inviteId int) string {
	csrfToken, ok := fc.getCookieValue(&user.Client, fc.BaseURL+"/u/home", "csrf_token")
	if !ok {
		panic("no csrf_token cookie available for authenticated request")
	}

	form := url.Values{}
	form.Add("inviteId", strconv.Itoa(inviteId))
	form.Add("csrf_token", csrfToken)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/u/acceptInvite", fc.BaseURL), strings.NewReader(form.Encode()))
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

func (fc *FantasyClient) getInviteId(body string) (int, bool) {
	prefix := "name=\"inviteId\" value=\""
	if strings.Count(body, prefix) == 0 {
		return -1, false
	}

	idx := strings.Index(body, prefix) + len(prefix)
	sliced := body[idx:]
	endIdx := strings.Index(sliced, "\"")
	id, err := strconv.Atoi(sliced[:endIdx])

	if err != nil {
		panic(err)
	}

	return id, true
}

// StartDraft transitions the draft to picking.
func (fc *FantasyClient) StartDraft(user *User, draftId int) {
	slog.Info("Start Draft", "Draft Id", draftId, "User", user.Username)

	csrfToken, ok := fc.getCookieValue(&user.Client, fc.BaseURL+"/u/home", "csrf_token")
	if !ok {
		panic("no csrf_token cookie available for authenticated request")
	}

	form := url.Values{}
	form.Add("csrf_token", csrfToken)

	resp, err := user.Client.Post(fmt.Sprintf("%s/u/draft/%d/startDraft", fc.BaseURL, draftId), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("Failed to start draft", "Draft Id", draftId, "Username", user.Username, "Error", err)
		panic(err)
	}

	if resp.StatusCode != 200 {
		slog.Error("Failed to start draft", "Draft Id", draftId, "User", user.Username)
		panic("failed to start draft")
	}

	slog.Info("Start Draft Request Made", "Draft Id", draftId, "User", user.Username, "Status", resp.StatusCode)
	slog.Info("Started Draft", "Draft Id", draftId)
}

// WaitUntilDraftState polls the draft status until it matches the requested
// status or the timeout is reached.
func (fc *FantasyClient) WaitUntilDraftState(user *User, draftId int, requestedStatus string, timeout time.Duration) {
	waitTime := 30 * time.Second
	timeoutTime := time.Now().Add(timeout)
	currentStatus := fc.GetCurrentDraftStatus(user, draftId)
	for currentStatus != requestedStatus {
		slog.Info("Checking if current draft is in requested status", "Requested Status", requestedStatus, "Current Status", currentStatus)
		if time.Now().After(timeoutTime) {
			panic("wait until draft state timeout reached")
		}

		currentStatus = fc.GetCurrentDraftStatus(user, draftId)
		time.Sleep(waitTime)
	}
}

// GetCurrentDraftStatus returns the current status string for the draft.
func (fc *FantasyClient) GetCurrentDraftStatus(user *User, draftId int) string {
	profilePage := fc.getDraftIdPage(user, draftId)
	return fc.parseDraftStatus(profilePage)
}

func (fc *FantasyClient) getDraftIdPage(user *User, draftId int) string {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/u/draft/%d/profile", fc.BaseURL, draftId), nil)
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

func (fc *FantasyClient) parseDraftStatus(profilePage string) string {
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

// IsPickingPlayer returns true if it is the user's turn to pick.
func (fc *FantasyClient) IsPickingPlayer(user *User, draftId int) bool {
	slog.Info("Getting picking player", "Draft Id", draftId, "User", user.Username)

	resp, err := user.Client.Get(fmt.Sprintf("%s/u/draft/%d/pick", fc.BaseURL, draftId))
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

	return strings.Contains(string(body), `name="pickInput"`) && !strings.Contains(string(body), `disabled name="pickInput"`)
}

// MakePickRequest attempts to pick the given team. It returns (true, "") on
// success and (false, errorMessage) when the server rejects the pick.
func (fc *FantasyClient) MakePickRequest(draftId int, user *User, team int) (bool, string) {
	slog.Info("Make Pick", "Draft Id", draftId, "User", user.Username, "Team", team)

	csrfToken, ok := fc.getCookieValue(&user.Client, fc.BaseURL+"/u/home", "csrf_token")
	if !ok {
		panic("no csrf_token cookie available for authenticated request")
	}

	form := url.Values{}
	form.Add("pickInput", strconv.Itoa(team))
	form.Add("csrf_token", csrfToken)

	resp, err := user.Client.Post(fmt.Sprintf("%s/u/draft/%d/makePick", fc.BaseURL, draftId), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
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

	bodyStr := string(body)
	hasError := strings.Contains(bodyStr, "id=\"pickError\"")

	if hasError {
		doc, err := html.Parse(strings.NewReader(bodyStr))
		if err != nil {
			return false, ""
		}
		errorDiv := FindNodeById(doc, "pickError")
		if errorDiv != nil {
			return false, ExtractText(errorDiv)
		}
		return false, ""
	}

	return true, ""
}

// LoadValidTeams reads a newline-delimited file of integer team numbers.
func (fc *FantasyClient) LoadValidTeams(path string) []int {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	var validTeams []int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		teamNum, err := strconv.Atoi(line)
		if err != nil {
			panic(err)
		}
		validTeams = append(validTeams, teamNum)
	}
	return validTeams
}

// CreateRandomString returns a random lowercase string of length between
// minLen and maxLen.
func (fc *FantasyClient) CreateRandomString(minLen int, maxLen int) string {
	alphabet := "abcdefghijklmnopqrstuvwxyz"
	length := rand.IntN(maxLen-minLen) + minLen
	var sb strings.Builder
	for range length {
		sb.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return sb.String()
}

func (fc *FantasyClient) getCookieValue(client *http.Client, targetUrl string, name string) (string, bool) {
	u, _ := url.Parse(targetUrl)
	for _, c := range client.Jar.Cookies(u) {
		if c.Name == name {
			return c.Value, true
		}
	}
	return "", false
}

// HTML parsing helpers.

func FindNodeById(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == "id" && attr.Val == id {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := FindNodeById(c, id); found != nil {
			return found
		}
	}
	return nil
}

func FindNodesByClass(n *html.Node, classParts ...string) []*html.Node {
	var results []*html.Node
	if n.Type == html.ElementNode {
		class := GetAttribute(n, "class")
		if class != "" && ContainsAll(class, classParts...) {
			results = append(results, n)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		results = append(results, FindNodesByClass(c, classParts...)...)
	}
	return results
}

func FindNodeByClass(n *html.Node, classParts ...string) *html.Node {
	if n.Type == html.ElementNode {
		class := GetAttribute(n, "class")
		if class != "" && ContainsAll(class, classParts...) {
			return n
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := FindNodeByClass(c, classParts...); found != nil {
			return found
		}
	}
	return nil
}

func FindNodeByTagAndClass(n *html.Node, tag string, classParts ...string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		class := GetAttribute(n, "class")
		if class != "" && ContainsAll(class, classParts...) {
			return n
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := FindNodeByTagAndClass(c, tag, classParts...); found != nil {
			return found
		}
	}
	return nil
}

func GetAttribute(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func ExtractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(ExtractText(c))
	}
	return sb.String()
}

func ContainsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
