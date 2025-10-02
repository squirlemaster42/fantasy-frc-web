package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHandleViewDraftProfile_InvalidDraftID(t *testing.T) {
	// Setup test helper
	th := NewTestHelper(t)
	defer th.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/u/draft/invalid/profile", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Mock authenticated user
	th.MockUserBySessionToken("valid-token", "550e8400-e29b-41d4-a716-446655440000", "testuser")
	cookie := &http.Cookie{Name: "sessionToken", Value: "valid-token"}
	req.AddCookie(cookie)

	handler := th.CreateMockHandler()

	// Execute
	err := handler.HandleViewDraftProfile(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	// Verify error message is user-friendly, not exposing internal details
	assert.Contains(t, rec.Body.String(), "Invalid draft ID")

	th.AssertExpectations(t)
}

func TestHandleUpdateDraftProfile_XSSAttempt(t *testing.T) {
	// Setup test helper
	th := NewTestHelper(t)
	defer th.Close()

	// Setup form with XSS payload and required fields
	f := make(url.Values)
	f.Set("draftName", "<script>alert('xss')</script>")
	f.Set("description", "valid description")
	f.Set("interval", "30")
	f.Set("startTime", "2024-01-01T10:00")
	f.Set("endTime", "2024-01-02T10:00")

	req := httptest.NewRequest(http.MethodPost, "/u/draft/1/update", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Mock authenticated user as draft owner
	th.MockUserBySessionToken("owner-token", "550e8400-e29b-41d4-a716-446655440001", "owner")
	th.MockDraftRetrieval(1, true, "550e8400-e29b-41d4-a716-446655440001")
	th.MockDraftUpdate(1, false) // Update will fail due to XSS validation

	cookie := &http.Cookie{Name: "sessionToken", Value: "owner-token"}
	req.AddCookie(cookie)
	c.SetParamNames("id")
	c.SetParamValues("1")

	handler := th.CreateMockHandler()

	// Execute
	err := handler.HandleUpdateDraftProfile(c)

	// Assert
	assert.NoError(t, err)
	// Should return validation error for XSS attempt
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid characters")

	th.AssertExpectations(t)
}

func TestHandleUpdateDraftProfile_UnauthorizedUser(t *testing.T) {
	// Setup test helper
	th := NewTestHelper(t)
	defer th.Close()

	// Setup valid form data
	f := make(url.Values)
	f.Set("draftName", "Test Draft")
	f.Set("description", "Test description")
	f.Set("interval", "30")

	req := httptest.NewRequest(http.MethodPost, "/u/draft/1/update", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Mock authenticated user who is NOT the draft owner
	th.MockUserBySessionToken("not-owner-token", "550e8400-e29b-41d4-a716-446655440002", "notowner")
	th.MockDraftRetrieval(1, true, "550e8400-e29b-41d4-a716-446655440001") // Draft exists but user is not owner

	cookie := &http.Cookie{Name: "sessionToken", Value: "not-owner-token"}
	req.AddCookie(cookie)

	// Set path parameter
	c.SetParamNames("id")
	c.SetParamValues("1")

	handler := th.CreateMockHandler()

	// Execute
	err := handler.HandleUpdateDraftProfile(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "Permission denied")

	th.AssertExpectations(t)
}

func TestHandleUpdateDraftProfile_InvalidInterval(t *testing.T) {
	invalidIntervals := []string{"", "0", "-1", "9999", "not-a-number"}

	for _, interval := range invalidIntervals {
		t.Run(fmt.Sprintf("interval_%s", interval), func(t *testing.T) {
			// Setup form with invalid interval
			f := make(url.Values)
			f.Set("draftName", "Test Draft")
			f.Set("description", "Test description")
			f.Set("interval", interval)

			req := httptest.NewRequest(http.MethodPost, "/u/draft/1/update", strings.NewReader(f.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			// Mock authenticated user as draft owner
			cookie := &http.Cookie{Name: "sessionToken", Value: "owner-token"}
			req.AddCookie(cookie)

			// Set path parameter
			c.SetParamNames("id")
			c.SetParamValues("1")

			handler := &Handler{
				// Database would be mocked here
			}

			// Execute
			err := handler.HandleUpdateDraftProfile(c)

			// Assert proper validation error
			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "interval")
		})
	}
}

func TestHandleUpdateDraftProfile_EmptyDraftName(t *testing.T) {
	// Setup form with empty draft name
	f := make(url.Values)
	f.Set("draftName", "")
	f.Set("description", "valid description")
	f.Set("interval", "30")

	req := httptest.NewRequest(http.MethodPost, "/u/draft/1/update", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Mock authenticated user as draft owner
	cookie := &http.Cookie{Name: "sessionToken", Value: "owner-token"}
	req.AddCookie(cookie)

	// Set path parameter
	c.SetParamNames("id")
	c.SetParamValues("1")

	handler := &Handler{
		// Database would be mocked here
	}

	// Execute
	err := handler.HandleUpdateDraftProfile(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "draft name cannot be empty")
}

func TestHandleUpdateDraftProfile_DraftNameTooLong(t *testing.T) {
	// Setup form with draft name that's too long
	longName := strings.Repeat("a", 101)
	f := make(url.Values)
	f.Set("draftName", longName)
	f.Set("description", "valid description")
	f.Set("interval", "30")

	req := httptest.NewRequest(http.MethodPost, "/u/draft/1/update", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Mock authenticated user as draft owner
	cookie := &http.Cookie{Name: "sessionToken", Value: "owner-token"}
	req.AddCookie(cookie)

	// Set path parameter
	c.SetParamNames("id")
	c.SetParamValues("1")

	handler := &Handler{
		// Database would be mocked here
	}

	// Execute
	err := handler.HandleUpdateDraftProfile(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "draft name must be no more than 100 characters")
}

func TestHandleStartDraft_UnauthenticatedUser(t *testing.T) {
	// Setup request without session cookie
	req := httptest.NewRequest(http.MethodPost, "/u/draft/1/startDraft", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Set path parameter
	c.SetParamNames("id")
	c.SetParamValues("1")

	handler := &Handler{
		// Database would be mocked here
	}

	// Execute
	err := handler.HandleStartDraft(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleStartDraft_InvalidDraftID(t *testing.T) {
	// Setup request with invalid draft ID
	req := httptest.NewRequest(http.MethodPost, "/u/draft/abc/startDraft", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Mock authenticated user
	cookie := &http.Cookie{Name: "sessionToken", Value: "valid-token"}
	req.AddCookie(cookie)

	// Set path parameter
	c.SetParamNames("id")
	c.SetParamValues("abc")

	handler := &Handler{
		// Database would be mocked here
	}

	// Execute
	err := handler.HandleStartDraft(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Draft Id is not a number")
}

func TestSearchPlayers_UnauthenticatedUser(t *testing.T) {
	// Setup search request without authentication
	f := make(url.Values)
	f.Set("search", "test")

	req := httptest.NewRequest(http.MethodPost, "/u/draft/1/searchPlayers", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Set path parameter
	c.SetParamNames("id")
	c.SetParamValues("1")

	handler := &Handler{
		// Database would be mocked here
	}

	// Execute
	err := handler.SearchPlayers(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestInviteDraftPlayer_InvalidUserUUID(t *testing.T) {
	// Setup invite request with invalid UUID
	f := make(url.Values)
	f.Set("userUuid", "invalid-uuid")

	req := httptest.NewRequest(http.MethodPost, "/u/draft/1/invitePlayer", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Mock authenticated user
	cookie := &http.Cookie{Name: "sessionToken", Value: "valid-token"}
	req.AddCookie(cookie)

	// Set path parameter
	c.SetParamNames("id")
	c.SetParamValues("1")

	handler := &Handler{
		// Database would be mocked here
	}

	// Execute
	err := handler.InviteDraftPlayer(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid UUID")
}
