package handler

import (
	"sync"
	"time"

	"server/config"
)

const (
	// Domain constants.
	teamPrefix           = "frc"
	htmxCurrentUrlHeader = "Hx-Current-Url"
	webhookSecretFileMode = 0600
)

var (
	defaultsOnce sync.Once
	defaults     handlerDefaults
)

type handlerDefaults struct {
	LeaderboardPerPage            int           `env:"LEADERBOARD_PER_PAGE" default:"25"`
	WsReadBufferSize              int           `env:"WS_READ_BUFFER_SIZE" default:"1024"`
	WsWriteBufferSize             int           `env:"WS_WRITE_BUFFER_SIZE" default:"1024"`
	WsReadTimeout                 time.Duration `env:"WS_READ_TIMEOUT" default:"120s"`
	WsPingInterval                time.Duration `env:"WS_PING_INTERVAL" default:"30s"`
	WsWriteTimeout                time.Duration `env:"WS_WRITE_TIMEOUT" default:"10s"`
	TbaWebhookMaxBodyBytes        int64         `env:"TBA_WEBHOOK_MAX_BODY_BYTES" default:"1048576"`
	TbaUpcomingMatchTeamCount     int           `env:"TBA_UPCOMING_MATCH_TEAM_COUNT" default:"6"`
	AvatarHttpCacheMaxAgeSeconds  int           `env:"AVATAR_HTTP_CACHE_MAX_AGE_SECONDS" default:"604800"`
}

func loadDefaults() handlerDefaults {
	var d handlerDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *handlerDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

func LeaderboardPerPage() int                { return getDefaults().LeaderboardPerPage }
func WsReadBufferSize() int                  { return getDefaults().WsReadBufferSize }
func WsWriteBufferSize() int                 { return getDefaults().WsWriteBufferSize }
func WsReadTimeout() time.Duration           { return getDefaults().WsReadTimeout }
func WsPingInterval() time.Duration          { return getDefaults().WsPingInterval }
func WsWriteTimeout() time.Duration          { return getDefaults().WsWriteTimeout }
func TbaWebhookMaxBodyBytes() int64          { return getDefaults().TbaWebhookMaxBodyBytes }
func TbaUpcomingMatchTeamCount() int         { return getDefaults().TbaUpcomingMatchTeamCount }
func AvatarHttpCacheMaxAgeSeconds() int      { return getDefaults().AvatarHttpCacheMaxAgeSeconds }
