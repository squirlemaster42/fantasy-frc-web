package middleware

import (
	"server/utils"
	"sync"
	"time"
)

const (
	rateLimitLoginAttemptsEnvKey    = "RATE_LIMIT_LOGIN_ATTEMPTS"
	defaultRateLimitLoginAttempts   = 5
	rateLimitRegisterAttemptsEnvKey  = "RATE_LIMIT_REGISTER_ATTEMPTS"
	defaultRateLimitRegisterAttempts = 3
	rateLimitAuthWindowEnvKey       = "RATE_LIMIT_AUTH_WINDOW"
	defaultRateLimitAuthWindow      = 15 * time.Minute

	rateLimitRedisPingTimeoutEnvKey = "RATE_LIMIT_REDIS_PING_TIMEOUT"
	defaultRateLimitRedisPingTimeout = 2 * time.Second

	hstsMaxAgeSecondsEnvKey = "HSTS_MAX_AGE_SECONDS"
	defaultHstsMaxAgeSeconds = 63072000

	// Domain constants that are not operational tuning knobs.
	rateLimitKeyPrefixLogin    = "rate_limit:login"
	rateLimitKeyPrefixRegister = "rate_limit:register"
	rateLimitKeyPrefixGeneral  = "rate_limit:general"
	rateLimitGeneralWindow     = time.Minute
	redisProtocolVersion       = 2

	// CSRF cookie/form/header names for double-submit cookie pattern.
	CsrfCookieName      = "csrf_cookie"
	CsrfTokenFieldName  = "csrf_token"
	CsrfTokenHeaderName = "X-CSRF-Token"
	CsrfTokenLength     = 32
)

var (
	defaultsOnce sync.Once
	defaults     middlewareDefaults
)

type middlewareDefaults struct {
	rateLimitLoginAttempts   int64
	rateLimitRegisterAttempts int64
	rateLimitAuthWindow      time.Duration
	rateLimitRedisPingTimeout time.Duration
	hstsMaxAgeSeconds        int
}

func loadDefaults() middlewareDefaults {
	return middlewareDefaults{
		rateLimitLoginAttempts:    utils.MustGetEnvInt64(rateLimitLoginAttemptsEnvKey, defaultRateLimitLoginAttempts),
		rateLimitRegisterAttempts: utils.MustGetEnvInt64(rateLimitRegisterAttemptsEnvKey, defaultRateLimitRegisterAttempts),
		rateLimitAuthWindow:       utils.MustGetEnvDuration(rateLimitAuthWindowEnvKey, defaultRateLimitAuthWindow),
		rateLimitRedisPingTimeout: utils.MustGetEnvDuration(rateLimitRedisPingTimeoutEnvKey, defaultRateLimitRedisPingTimeout),
		hstsMaxAgeSeconds:         utils.MustGetEnvInt(hstsMaxAgeSecondsEnvKey, defaultHstsMaxAgeSeconds),
	}
}

func getDefaults() *middlewareDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// RateLimitLoginAttempts returns the max login attempts within the auth window.
func RateLimitLoginAttempts() int64 { return getDefaults().rateLimitLoginAttempts }

// RateLimitRegisterAttempts returns the max register attempts within the auth window.
func RateLimitRegisterAttempts() int64 { return getDefaults().rateLimitRegisterAttempts }

// RateLimitAuthWindow returns the sliding window for login/register rate limits.
func RateLimitAuthWindow() time.Duration { return getDefaults().rateLimitAuthWindow }

// RateLimitRedisPingTimeout returns the timeout for the Redis availability check.
func RateLimitRedisPingTimeout() time.Duration { return getDefaults().rateLimitRedisPingTimeout }

// HstsMaxAgeSeconds returns the Strict-Transport-Security max-age value.
func HstsMaxAgeSeconds() int { return getDefaults().hstsMaxAgeSeconds }
