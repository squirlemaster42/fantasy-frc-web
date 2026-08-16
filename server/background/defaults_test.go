package background

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults_DefaultValues(t *testing.T) {
	d := loadDefaults()

	assert.Equal(t, 2, d.cleanupSessionLeewayHours)
	assert.Equal(t, 60, d.cleanupIntervalMinutes)
	assert.Equal(t, 1*time.Minute, d.draftDaemonTickInterval)
	assert.Equal(t, 55*time.Second, d.draftDaemonTickTimeout)
}

func TestLoadDefaults_Override(t *testing.T) {
	t.Setenv("SESSION_CLEANUP_LEEWAY_HOURS", "4")
	t.Setenv("CLEANUP_INTERVAL_MINUTES", "30")
	t.Setenv("DRAFT_DAEMON_TICK_INTERVAL", "2m")
	t.Setenv("DRAFT_DAEMON_TICK_TIMEOUT", "30s")

	d := loadDefaults()

	assert.Equal(t, 4, d.cleanupSessionLeewayHours)
	assert.Equal(t, 30, d.cleanupIntervalMinutes)
	assert.Equal(t, 2*time.Minute, d.draftDaemonTickInterval)
	assert.Equal(t, 30*time.Second, d.draftDaemonTickTimeout)
}
