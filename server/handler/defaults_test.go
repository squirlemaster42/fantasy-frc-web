package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerDefaults(t *testing.T) {
	assert.Positive(t, LeaderboardPerPage())
	assert.Positive(t, WsReadBufferSize())
	assert.Positive(t, WsWriteBufferSize())
	assert.Positive(t, WsReadTimeout())
	assert.Positive(t, WsPingInterval())
	assert.Positive(t, WsWriteTimeout())
	assert.Positive(t, WsMaxMessageSize())
	assert.Positive(t, TbaWebhookMaxBodyBytes())
	assert.Positive(t, TbaUpcomingMatchTeamCount())
	assert.Positive(t, AvatarHttpCacheMaxAgeSeconds())
}
