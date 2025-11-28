package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

// Test utilities for CSRF testing
type CSRFTestClient struct {
	client   *http.Client
	baseURL  string
	jar      *cookiejar.Jar
}

type CSRFTestResult struct {
	Endpoint      string
	Method        string
	HasToken      bool
	ValidToken    bool
	MissingToken  bool
	InvalidToken  bool
	CrossOrigin   bool
	StatusCodes    map[string]int
	Passed        bool
}

func CreateCSRFTestClient(baseURL string) *CSRFTestClient {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout:  10 * time.Second,
	}
	
	return &CSRFTestClient{
		client:  client,
		baseURL: baseURL,
		jar:     jar,
	}
}

func (c *CSRFTestClient) extractCSRFToken(pageURL string) (string, error) {
	resp, err := c.client.Get(c.baseURL + pageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Parse HTML to find CSRF token
	tokenRegex := regexp.MustCompile(`name="csrf_token" value="([^"]*)"`)
	body := new(bytes.Buffer)
	body.ReadFrom(resp.Body)
	
	matches := tokenRegex.FindStringSubmatch(body.String())
	if len(matches) < 2 {
		return "", fmt.Errorf("CSRF token not found on page %s", pageURL)
	}
	
	return matches[1], nil
}

func (c *CSRFTestClient) makeRequest(method, endpoint, body, token string, headers map[string]string) (*http.Response, error) {
	var req *http.Request
	var err error
	
	if body != "" {
		req, err = http.NewRequest(method, c.baseURL+endpoint, strings.NewReader(body))
	} else {
		req, err = http.NewRequest(method, c.baseURL+endpoint, nil)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	// Add CSRF token if provided
	if token != "" {
		if body == "" {
			req.Header.Set("X-CSRF-Token", token)
		} else {
			body += "&csrf_token=" + token
			req.Body = io.NopCloser(strings.NewReader(body))
		}
	}
	
	return c.client.Do(req)
}

func (c *CSRFTestClient) testCSRFEndpoint(t *testing.T, endpoint, method, testData string) CSRFTestResult {
	result := CSRFTestResult{
		Endpoint:   endpoint,
		Method:     method,
		StatusCodes: make(map[string]int),
		Passed:     true,
	}
	
	// Test 1: Check if page has CSRF token (for GET endpoints)
	if method == "GET" {
		token, err := c.extractCSRFToken(endpoint)
		result.HasToken = (err == nil && token != "")
		if !result.HasToken {
			result.Passed = false
		}
	}
	
	// Get valid CSRF token for POST tests
	var validToken string
	if method == "POST" {
		// Try to get token from login page as fallback
		validToken, _ = c.extractCSRFToken("/login")
		if validToken == "" {
			// Try to get token from the endpoint itself if it's a form page
			validToken, _ = c.extractCSRFToken(endpoint)
		}
	}
	
	// Test 2: Valid CSRF token
	if validToken != "" {
		resp, err := c.makeRequest(method, endpoint, testData, validToken, nil)
		if err == nil {
			result.StatusCodes["valid_token"] = resp.StatusCode
			result.ValidToken = (resp.StatusCode == 200 || resp.StatusCode == 302 || resp.StatusCode == 201)
			resp.Body.Close()
		}
	}
	
	// Test 3: Missing CSRF token
	resp, err := c.makeRequest(method, endpoint, testData, "", nil)
	if err == nil {
		result.StatusCodes["missing_token"] = resp.StatusCode
		result.MissingToken = (resp.StatusCode == 403)
		if !result.MissingToken {
			result.Passed = false
		}
		resp.Body.Close()
	}
	
	// Test 4: Invalid CSRF token
	resp, err = c.makeRequest(method, endpoint, testData, "invalid_token", nil)
	if err == nil {
		result.StatusCodes["invalid_token"] = resp.StatusCode
		result.InvalidToken = (resp.StatusCode == 403)
		if !result.InvalidToken {
			result.Passed = false
		}
		resp.Body.Close()
	}
	
	// Test 5: Cross-origin request
	crossOriginHeaders := map[string]string{
		"Origin": "https://evil-site.com",
		"Referer": "https://evil-site.com/attack",
	}
	resp, err = c.makeRequest(method, endpoint, testData, validToken, crossOriginHeaders)
	if err == nil {
		result.StatusCodes["cross_origin"] = resp.StatusCode
		result.CrossOrigin = (resp.StatusCode == 403)
		if !result.CrossOrigin {
			result.Passed = false
		}
		resp.Body.Close()
	}
	
	return result
}

func TestCSRFProtection(t *testing.T) {
	// Load environment variables
	err := godotenv.Load("../.env")
	if err != nil {
		t.Skip("Skipping CSRF tests - .env file not found")
	}
	
	client := CreateCSRFTestClient("http://localhost:3000")
	
	// Test cases for all protected endpoints
	testCases := []struct {
		name      string
		endpoint  string
		method    string
		testData  string
		needsAuth bool
	}{
		{
			name:      "Login",
			endpoint:  "/login",
			method:    "POST",
			testData:  "username=test&password=test",
			needsAuth: false,
		},
		{
			name:      "Register",
			endpoint:  "/register",
			method:    "POST",
			testData:  "username=newuser&password=test123&confirmPassword=test123",
			needsAuth: false,
		},
		{
			name:      "CreateDraft",
			endpoint:  "/u/createDraft",
			method:    "POST",
			testData:  "displayName=Test Draft&description=Test Description",
			needsAuth: true,
		},
		{
			name:      "UpdateDraft",
			endpoint:  "/u/draft/1/updateDraft",
			method:    "POST",
			testData:  "displayName=Updated Draft",
			needsAuth: true,
		},
		{
			name:      "StartDraft",
			endpoint:  "/u/draft/1/startDraft",
			method:    "POST",
			testData:  "",
			needsAuth: true,
		},
		{
			name:      "MakePick",
			endpoint:  "/u/draft/1/makePick",
			method:    "POST",
			testData:  "pickInput=frc1234",
			needsAuth: true,
		},
		{
			name:      "InvitePlayer",
			endpoint:  "/u/draft/1/invitePlayer",
			method:    "POST",
			testData:  "username=testuser",
			needsAuth: true,
		},
		{
			name:      "AcceptInvite",
			endpoint:  "/u/acceptInvite",
			method:    "POST",
			testData:  "inviteId=1",
			needsAuth: true,
		},
		{
			name:      "TeamScore",
			endpoint:  "/u/team/score",
			method:    "POST",
			testData:  "teamNumber=1234",
			needsAuth: true,
		},
		{
			name:      "SearchPlayers",
			endpoint:  "/u/searchPlayers",
			method:    "POST",
			testData:  "search=test",
			needsAuth: true,
		},
		{
			name:      "SkipPickToggle",
			endpoint:  "/u/draft/1/skipPickToggle",
			method:    "POST",
			testData:  "skipping=true",
			needsAuth: true,
		},
	}
	
	// First, test webhook exemption (should not require CSRF)
	t.Run("WebhookCSRFExemption", func(t *testing.T) {
		resp, err := client.makeRequest("POST", "/tbaWebhook", `{"test":"data"}`, "", nil)
		if err != nil {
			t.Fatalf("Failed to make webhook request: %v", err)
		}
		defer resp.Body.Close()
		
		// Webhook should work without CSRF token
		assert.Equal(t, 200, resp.StatusCode, "Webhook should not require CSRF token")
	})
	
	// Test each endpoint
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip authentication-required tests if we can't authenticate
			if tc.needsAuth {
				// Try to authenticate first
				token, err := client.extractCSRFToken("/login")
				if err != nil {
					t.Skip("Cannot authenticate - skipping protected endpoint test")
				}
				
				// Login with test user (assuming test user exists)
				resp, err := client.makeRequest("POST", "/login", "username=test&password=test", token, nil)
				if err != nil {
					t.Skip("Cannot authenticate - skipping protected endpoint test")
				}
				resp.Body.Close()
			}
			
			result := client.testCSRFEndpoint(t, tc.endpoint, tc.method, tc.testData)
			
			// Assert results
			if tc.method == "GET" {
				assert.True(t, result.HasToken, "GET endpoint should have CSRF token in form")
			}
			
			assert.True(t, result.MissingToken, "Missing CSRF token should be rejected")
			assert.True(t, result.InvalidToken, "Invalid CSRF token should be rejected")
			assert.True(t, result.CrossOrigin, "Cross-origin request should be rejected")
			
			if result.Passed {
				t.Logf("✅ %s: CSRF protection working correctly", tc.name)
			} else {
				t.Errorf("❌ %s: CSRF protection failed", tc.name)
				t.Logf("Status codes: %+v", result.StatusCodes)
			}
		})
	}
}

func TestCSRFTokenUniqueness(t *testing.T) {
	client := CreateCSRFTestClient("http://localhost:3000")
	
	// Get CSRF tokens from multiple requests
	token1, err1 := client.extractCSRFToken("/login")
	token2, err2 := client.extractCSRFToken("/login")
	
	if err1 != nil || err2 != nil {
		t.Skip("Cannot extract CSRF tokens - skipping uniqueness test")
	}
	
	// Tokens should be different (session-based)
	assert.NotEqual(t, token1, token2, "CSRF tokens should be unique per session")
}

func TestCSRFTokenExpiration(t *testing.T) {
	client := CreateCSRFTestClient("http://localhost:3000")
	
	// Get a CSRF token
	token, err := client.extractCSRFToken("/login")
	if err != nil {
		t.Skip("Cannot extract CSRF token - skipping expiration test")
	}
	
	// Clear cookies to simulate expired session
	client.jar.SetCookies(&url.URL{Scheme: "http", Host: "localhost:3000"}, []*http.Cookie{})
	
	// Try to use the token with expired session
	resp, err := client.makeRequest("POST", "/login", "username=test&password=test", token, nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	// Should fail due to expired session
	assert.NotEqual(t, 200, resp.StatusCode, "Request with expired session should fail")
}