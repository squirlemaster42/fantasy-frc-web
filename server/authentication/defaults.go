package authentication

import (
	"sync"

	"server/config"
)

// SessionCookieName is the name of the session token cookie.
const SessionCookieName = "sessionToken"

var (
	defaultsOnce sync.Once
	defaults     authDefaults
)

type authDefaults struct {
	SessionTokenBytes int `env:"SESSION_TOKEN_BYTES" default:"16"`
}

func loadDefaults() authDefaults {
	var d authDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *authDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// SessionTokenBytes returns the number of random bytes used to generate a session token.
func SessionTokenBytes() int { return getDefaults().SessionTokenBytes }
