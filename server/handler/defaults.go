package handler

import (
	"server/utils"
	"sync"
	"time"
)

const (
	leaderboardPerPageEnvKey = "LEADERBOARD_PER_PAGE"
	defaultLeaderboardPerPage = 25

	wsReadBufferSizeEnvKey = "WS_READ_BUFFER_SIZE"
	defaultWsReadBufferSize = 1024

	wsWriteBufferSizeEnvKey = "WS_WRITE_BUFFER_SIZE"
	defaultWsWriteBufferSize = 1024

	wsReadTimeoutEnvKey = "WS_READ_TIMEOUT"
	defaultWsReadTimeout = 120 * time.Second

	wsPingIntervalEnvKey = "WS_PING_INTERVAL"
	defaultWsPingInterval = 30 * time.Second

	wsWriteTimeoutEnvKey = "WS_WRITE_TIMEOUT"
	defaultWsWriteTimeout = 10 * time.Second

	tbaWebhookMaxBodyBytesEnvKey = "TBA_WEBHOOK_MAX_BODY_BYTES"
	defaultTbaWebhookMaxBodyBytes = 1 << 20 // 1 MB

	tbaUpcomingMatchTeamCountEnvKey = "TBA_UPCOMING_MATCH_TEAM_COUNT"
	defaultTbaUpcomingMatchTeamCount = 6

	avatarHttpCacheMaxAgeSecondsEnvKey = "AVATAR_HTTP_CACHE_MAX_AGE_SECONDS"
	defaultAvatarHttpCacheMaxAgeSeconds = 604800 // 7 days

	// Domain constants.
	teamPrefix          = "frc"
	htmxCurrentUrlHeader = "Hx-Current-Url"
	webhookSecretFileMode = 0600
)

var (
	defaultsOnce sync.Once
	defaults     handlerDefaults
)

type handlerDefaults struct {
	leaderboardPerPage         int
	wsReadBufferSize           int
	wsWriteBufferSize          int
	wsReadTimeout              time.Duration
	wsPingInterval             time.Duration
	wsWriteTimeout             time.Duration
	tbaWebhookMaxBodyBytes     int64
	tbaUpcomingMatchTeamCount  int
	avatarHttpCacheMaxAgeSeconds int
}

func loadDefaults() handlerDefaults {
	return handlerDefaults{
		leaderboardPerPage:         utils.MustGetEnvInt(leaderboardPerPageEnvKey, defaultLeaderboardPerPage),
		wsReadBufferSize:           utils.MustGetEnvInt(wsReadBufferSizeEnvKey, defaultWsReadBufferSize),
		wsWriteBufferSize:          utils.MustGetEnvInt(wsWriteBufferSizeEnvKey, defaultWsWriteBufferSize),
		wsReadTimeout:              utils.MustGetEnvDuration(wsReadTimeoutEnvKey, defaultWsReadTimeout),
		wsPingInterval:             utils.MustGetEnvDuration(wsPingIntervalEnvKey, defaultWsPingInterval),
		wsWriteTimeout:             utils.MustGetEnvDuration(wsWriteTimeoutEnvKey, defaultWsWriteTimeout),
		tbaWebhookMaxBodyBytes:     utils.MustGetEnvInt64(tbaWebhookMaxBodyBytesEnvKey, defaultTbaWebhookMaxBodyBytes),
		tbaUpcomingMatchTeamCount:  utils.MustGetEnvInt(tbaUpcomingMatchTeamCountEnvKey, defaultTbaUpcomingMatchTeamCount),
		avatarHttpCacheMaxAgeSeconds: utils.MustGetEnvInt(avatarHttpCacheMaxAgeSecondsEnvKey, defaultAvatarHttpCacheMaxAgeSeconds),
	}
}

func getDefaults() *handlerDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

func LeaderboardPerPage() int                { return getDefaults().leaderboardPerPage }
func WsReadBufferSize() int                  { return getDefaults().wsReadBufferSize }
func WsWriteBufferSize() int                 { return getDefaults().wsWriteBufferSize }
func WsReadTimeout() time.Duration           { return getDefaults().wsReadTimeout }
func WsPingInterval() time.Duration          { return getDefaults().wsPingInterval }
func WsWriteTimeout() time.Duration          { return getDefaults().wsWriteTimeout }
func TbaWebhookMaxBodyBytes() int64          { return getDefaults().tbaWebhookMaxBodyBytes }
func TbaUpcomingMatchTeamCount() int         { return getDefaults().tbaUpcomingMatchTeamCount }
func AvatarHttpCacheMaxAgeSeconds() int      { return getDefaults().avatarHttpCacheMaxAgeSeconds }
