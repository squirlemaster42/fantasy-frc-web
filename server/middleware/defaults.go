package middleware

import (
	"sync"
	"time"

	"server/config"
)

const (
	// Domain constants that are not operational tuning knobs.
	rateLimitKeyPrefixLogin    = "rate_limit:login"
	rateLimitKeyPrefixRegister = "rate_limit:register"
	rateLimitKeyPrefixGeneral = "rate_limit:general"
	rateLimitGeneralWindow    = time.Minute

	// CSRF cookie/form/header names for double-submit cookie pattern.
	CsrfCookieName      = "csrf_cookie"
	CsrfTokenFieldName  = "csrf_token"
	CsrfTokenHeaderName = "X-CSRF-Token" //nolint:gosec // HTTP header name, not a credential
	CsrfTokenLength     = 32
)

var (
	defaultsOnce sync.Once
	defaults     middlewareDefaults
)

type middlewareDefaults struct {
	RateLimitLoginAttempts    int64         `env:"RATE_LIMIT_LOGIN_ATTEMPTS" default:"5"`
	RateLimitRegisterAttempts int64         `env:"RATE_LIMIT_REGISTER_ATTEMPTS" default:"3"`
	RateLimitAuthWindow       time.Duration `env:"RATE_LIMIT_AUTH_WINDOW" default:"15m"`
	RateLimitRedisPingTimeout time.Duration `env:"RATE_LIMIT_REDIS_PING_TIMEOUT" default:"2s"`
	HstsMaxAgeSeconds         int           `env:"HSTS_MAX_AGE_SECONDS" default:"63072000"`
}

func loadDefaults() middlewareDefaults {
	var d middlewareDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *middlewareDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// RateLimitLoginAttempts returns the max login attempts within the auth window.
func RateLimitLoginAttempts() int64 { return getDefaults().RateLimitLoginAttempts }

// RateLimitRegisterAttempts returns the max register attempts within the auth window.
func RateLimitRegisterAttempts() int64 { return getDefaults().RateLimitRegisterAttempts }

// RateLimitAuthWindow returns the sliding window for login/register rate limits.
func RateLimitAuthWindow() time.Duration { return getDefaults().RateLimitAuthWindow }

// RateLimitRedisPingTimeout returns the timeout for the Redis availability check.
func RateLimitRedisPingTimeout() time.Duration { return getDefaults().RateLimitRedisPingTimeout }

// HstsMaxAgeSeconds returns the Strict-Transport-Security max-age value.
func HstsMaxAgeSeconds() int { return getDefaults().HstsMaxAgeSeconds }
