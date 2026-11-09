package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults_DefaultValues(t *testing.T) {
	d := loadDefaults()

	assert.Equal(t, int64(5), d.RateLimitLoginAttempts)
	assert.Equal(t, int64(3), d.RateLimitRegisterAttempts)
	assert.Equal(t, 15*time.Minute, d.RateLimitAuthWindow)
	assert.Equal(t, 2*time.Second, d.RateLimitRedisPingTimeout)
	assert.Equal(t, 63072000, d.HstsMaxAgeSeconds)
}

func TestLoadDefaults_Override(t *testing.T) {
	t.Setenv("RATE_LIMIT_LOGIN_ATTEMPTS", "10")
	t.Setenv("RATE_LIMIT_REGISTER_ATTEMPTS", "6")
	t.Setenv("RATE_LIMIT_AUTH_WINDOW", "30m")
	t.Setenv("RATE_LIMIT_REDIS_PING_TIMEOUT", "5s")
	t.Setenv("HSTS_MAX_AGE_SECONDS", "86400")

	d := loadDefaults()

	assert.Equal(t, int64(10), d.RateLimitLoginAttempts)
	assert.Equal(t, int64(6), d.RateLimitRegisterAttempts)
	assert.Equal(t, 30*time.Minute, d.RateLimitAuthWindow)
	assert.Equal(t, 5*time.Second, d.RateLimitRedisPingTimeout)
	assert.Equal(t, 86400, d.HstsMaxAgeSeconds)
}
