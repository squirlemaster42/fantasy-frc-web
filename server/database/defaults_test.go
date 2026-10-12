package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults_DefaultValues(t *testing.T) {
	d := loadDefaults()

	assert.Equal(t, 90, d.maxOpenConns)
	assert.Equal(t, 25, d.maxIdleConns)
	assert.Equal(t, 30*time.Minute, d.connMaxLifetime)
}

func TestLoadDefaults_Override(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "10")
	t.Setenv("DB_MAX_IDLE_CONNS", "5")
	t.Setenv("DB_CONN_MAX_LIFETIME", "5m")

	d := loadDefaults()

	assert.Equal(t, 10, d.maxOpenConns)
	assert.Equal(t, 5, d.maxIdleConns)
	assert.Equal(t, 5*time.Minute, d.connMaxLifetime)
}
