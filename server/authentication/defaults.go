package authentication

import (
	"server/utils"
	"sync"
)

const (
	sessionTokenBytesEnvKey = "SESSION_TOKEN_BYTES"
	defaultSessionTokenBytes = 16

	// SessionCookieName is the name of the session token cookie.
	SessionCookieName = "sessionToken"
)

var (
	defaultsOnce sync.Once
	defaults     authDefaults
)

type authDefaults struct {
	sessionTokenBytes int
}

func loadDefaults() authDefaults {
	return authDefaults{
		sessionTokenBytes: utils.MustGetEnvInt(sessionTokenBytesEnvKey, defaultSessionTokenBytes),
	}
}

func getDefaults() *authDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// SessionTokenBytes returns the number of random bytes used to generate a session token.
func SessionTokenBytes() int { return getDefaults().sessionTokenBytes }
