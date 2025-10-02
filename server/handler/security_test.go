package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestXSSPreventionInTemplates(t *testing.T) {
	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"javascript:alert('xss')",
		"<iframe src='javascript:alert(\"xss\")'>",
		"<svg onload=alert('xss')>",
		"<div onmouseover=alert('xss')>",
		"'><script>alert('xss')</script>",
		"\\\"><script>alert('xss')</script>",
	}

	for _, payload := range xssPayloads {
		t.Run(payload[:min(20, len(payload))], func(t *testing.T) {
			// Test draft names - simulate creating a draft with XSS payload
			f := make(url.Values)
			f.Set("draftName", payload)
			f.Set("description", "valid description")
			f.Set("interval", "30")

			req := httptest.NewRequest(http.MethodPost, "/u/draft/1/update", strings.NewReader(f.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			// Mock authenticated user
			cookie := &http.Cookie{Name: "sessionToken", Value: "owner-token"}
			req.AddCookie(cookie)
			c.SetParamNames("id")
			c.SetParamValues("1")

			handler := &Handler{
				// Database would be mocked here
			}

			// Execute
			err := handler.HandleUpdateDraftProfile(c)

			// Assert
			assert.NoError(t, err)

			// Check that XSS payload is not present in raw form
			body := rec.Body.String()
			assert.NotContains(t, body, payload, "XSS payload should be sanitized")

			// For script tags specifically, ensure they're escaped
			if strings.Contains(payload, "<script>") {
				assert.Contains(t, body, "&lt;script&gt;", "Script tags should be HTML escaped")
			}
		})
	}
}

func TestInputSanitization(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldAllow bool
		description string
	}{
		{
			name:        "normal text",
			input:       "Hello World",
			shouldAllow: true,
			description: "Normal text should be allowed",
		},
		{
			name:        "html entities",
			input:       "Tom & Jerry",
			shouldAllow: true,
			description: "HTML entities should be escaped",
		},
		{
			name:        "quotes",
			input:       `"Hello"`,
			shouldAllow: true,
			description: "Quotes should be escaped",
		},
		{
			name:        "script tag",
			input:       "<script>",
			shouldAllow: false,
			description: "Script tags should be rejected",
		},
		{
			name:        "img tag",
			input:       "<img src=x>",
			shouldAllow: false,
			description: "Image tags should be rejected",
		},
		{
			name:        "event handler",
			input:       "<div onclick=alert(1)>",
			shouldAllow: false,
			description: "Event handlers should be rejected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test draft name validation
			f := make(url.Values)
			f.Set("draftName", tc.input)
			f.Set("description", "valid description")
			f.Set("interval", "30")

			req := httptest.NewRequest(http.MethodPost, "/u/draft/1/update", strings.NewReader(f.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			// Mock authenticated user
			cookie := &http.Cookie{Name: "sessionToken", Value: "owner-token"}
			req.AddCookie(cookie)
			c.SetParamNames("id")
			c.SetParamValues("1")

			handler := &Handler{
				// Database would be mocked here
			}

			// Execute
			err := handler.HandleUpdateDraftProfile(c)

			// Assert
			assert.NoError(t, err)

			if tc.shouldAllow {
				// Should not get validation error for allowed input
				assert.NotEqual(t, http.StatusBadRequest, rec.Code,
					"Input should be allowed: %s", tc.description)
			} else {
				// Should get validation error for dangerous input
				assert.Equal(t, http.StatusBadRequest, rec.Code,
					"Input should be rejected: %s", tc.description)
				assert.Contains(t, rec.Body.String(), "invalid characters")
			}
		})
	}
}

func TestSQLInjectionPrevention(t *testing.T) {
	// Test that SQL injection attempts are properly handled
	sqlInjectionPayloads := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"' UNION SELECT * FROM users --",
		"admin'--",
		"1; SELECT * FROM users;",
	}

	for _, payload := range sqlInjectionPayloads {
		t.Run(payload[:min(20, len(payload))], func(t *testing.T) {
			// Test in username field during registration
			f := make(url.Values)
			f.Set("username", payload)
			f.Set("password", "ValidPass123!")
			f.Set("confirmPassword", "ValidPass123!")

			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(f.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			handler := &Handler{
				// Database would be mocked here
			}

			// Execute
			err := handler.HandlerRegisterPost(c)

			// Assert
			assert.NoError(t, err)

			// Should reject invalid username characters
			if strings.Contains(payload, " ") || strings.Contains(payload, ";") ||
				strings.Contains(payload, "'") || strings.Contains(payload, "--") {
				assert.Equal(t, http.StatusBadRequest, rec.Code,
					"SQL injection attempt should be rejected")
			}
		})
	}
}

func TestHTMXHeaderValidation(t *testing.T) {
	// Test that HTMX headers are validated if used for security
	req := httptest.NewRequest(http.MethodPost, "/u/draft/1/makePick", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.com/u/draft/1/pick")

	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Mock authenticated user
	cookie := &http.Cookie{Name: "sessionToken", Value: "valid-token"}
	req.AddCookie(cookie)
	c.SetParamNames("id")
	c.SetParamValues("1")

	handler := &Handler{
		// Database would be mocked here
	}

	// Execute pick request
	err := handler.HandlerPickRequest(c)

	// Assert
	assert.NoError(t, err)
	// Should process HTMX request normally
	// In a real implementation, we'd validate HX-Current-URL matches expected pattern
}

func TestContentTypeValidation(t *testing.T) {
	// Test that only expected content types are accepted
	testCases := []struct {
		contentType string
		shouldAllow bool
	}{
		{echo.MIMEApplicationForm, true},
		{echo.MIMEApplicationJSON, false},
		{"text/plain", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.contentType, func(t *testing.T) {
			f := make(url.Values)
			f.Set("draftName", "Test Draft")
			f.Set("description", "Test description")
			f.Set("interval", "30")

			req := httptest.NewRequest(http.MethodPost, "/u/draft/1/update", strings.NewReader(f.Encode()))
			if tc.contentType != "" {
				req.Header.Set(echo.HeaderContentType, tc.contentType)
			}
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			// Mock authenticated user
			cookie := &http.Cookie{Name: "sessionToken", Value: "owner-token"}
			req.AddCookie(cookie)
			c.SetParamNames("id")
			c.SetParamValues("1")

			handler := &Handler{
				// Database would be mocked here
			}

			// Execute
			err := handler.HandleUpdateDraftProfile(c)

			// Assert
			assert.NoError(t, err)

			if tc.shouldAllow {
				// Should process form data
				assert.NotEqual(t, http.StatusBadRequest, rec.Code)
			} else {
				// Should reject invalid content type
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestPathTraversalPrevention(t *testing.T) {
	// Test that path traversal attempts are prevented
	traversalPayloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"....//....//....//etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc/passwd",
	}

	for _, payload := range traversalPayloads {
		t.Run(payload[:min(20, len(payload))], func(t *testing.T) {
			// Test in any field that might be used in file operations
			// For now, test in draft name (though it might not be used for files)
			f := make(url.Values)
			f.Set("draftName", payload)
			f.Set("description", "valid description")
			f.Set("interval", "30")

			req := httptest.NewRequest(http.MethodPost, "/u/draft/1/update", strings.NewReader(f.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			// Mock authenticated user
			cookie := &http.Cookie{Name: "sessionToken", Value: "owner-token"}
			req.AddCookie(cookie)
			c.SetParamNames("id")
			c.SetParamValues("1")

			handler := &Handler{
				// Database would be mocked here
			}

			// Execute
			err := handler.HandleUpdateDraftProfile(c)

			// Assert
			assert.NoError(t, err)

			// Should reject path traversal characters
			if strings.Contains(payload, "..") || strings.Contains(payload, "/") || strings.Contains(payload, "\\") {
				assert.Equal(t, http.StatusBadRequest, rec.Code,
					"Path traversal attempt should be rejected")
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
