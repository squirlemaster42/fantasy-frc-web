package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults_DefaultValues(t *testing.T) {
	d := loadDefaults()

	assert.Equal(t, []string{"arc", "cur", "dal", "gal", "hop", "joh", "mil", "new", "cmptx"}, d.EventCodes)
	assert.Equal(t, "./webhookSecret.txt", d.WebhookSecretFile)
}

func TestLoadDefaults_EventCodesOverride(t *testing.T) {
	t.Setenv("TBA_EVENT_CODES", "foo,bar")

	d := loadDefaults()

	assert.Equal(t, []string{"foo", "bar"}, d.EventCodes)
}

func TestLoadDefaults_WebhookSecretFileOverride(t *testing.T) {
	t.Setenv("TBA_WEBHOOK_SECRET_FILE", "/tmp/secret.txt")

	d := loadDefaults()

	assert.Equal(t, "/tmp/secret.txt", d.WebhookSecretFile)
}
