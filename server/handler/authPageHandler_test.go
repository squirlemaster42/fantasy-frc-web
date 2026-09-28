package handler

import (
	"net/http"
	"testing"

	"server/authentication"
	"server/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetSessionCookie(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")

	h := &Handler{
		Config: ConfigGroup{
			SecureHttpCookie: true,
		},
	}

	sessionToken := "test-session-token"
	h.setSessionCookie(c, sessionToken)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, authentication.SessionCookieName, cookie.Name)
	assert.Equal(t, sessionToken, cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)
	assert.Equal(t, model.SessionExpirationDays()*86400, cookie.MaxAge)
}

func TestSetSessionCookie_InsecureWhenConfigured(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/login", "", "")

	h := &Handler{
		Config: ConfigGroup{
			SecureHttpCookie: false,
		},
	}

	h.setSessionCookie(c, "token")

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.False(t, cookies[0].Secure)
}
