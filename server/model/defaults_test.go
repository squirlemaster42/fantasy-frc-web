package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults_DefaultValues(t *testing.T) {
	d := loadDefaults()

	assert.Equal(t, 10, d.sessionExpirationDays)
}

func TestLoadDefaults_Override(t *testing.T) {
	t.Setenv("SESSION_EXPIRATION_DAYS", "30")

	d := loadDefaults()

	assert.Equal(t, 30, d.sessionExpirationDays)
}
