package cache

import (
	"strings"
	"sync"
	"time"

	"server/utils"
)

const (
	tbaPickCacheTypeEnvKey             = "TBA_PICK_CACHE_TYPE"
	defaultTbaPickCacheType              = "memory"

	tbaPickCacheRefreshIntervalEnvKey  = "TBA_PICK_CACHE_REFRESH_INTERVAL"
	defaultTbaPickCacheRefreshInterval = time.Hour

	tbaPickCacheRedisDBEnvKey          = "TBA_PICK_CACHE_REDIS_DB"
	defaultTbaPickCacheRedisDB         = 3

	tbaPickCacheRedisKeyPrefixEnvKey   = "TBA_PICK_CACHE_REDIS_KEY_PREFIX"
	defaultTbaPickCacheRedisKeyPrefix  = "tba:pick_validation:"
)

var (
	eventTeamDefaultsOnce sync.Once
	eventTeamDefaults     eventTeamCacheDefaults
)

type eventTeamCacheDefaults struct {
	cacheType       string
	refreshInterval time.Duration
	redisDB         int
	redisKeyPrefix  string
}

func loadEventTeamCacheDefaults() eventTeamCacheDefaults {
	return eventTeamCacheDefaults{
		cacheType:       utils.MustGetEnvString(tbaPickCacheTypeEnvKey, defaultTbaPickCacheType),
		refreshInterval: utils.MustGetEnvDuration(tbaPickCacheRefreshIntervalEnvKey, defaultTbaPickCacheRefreshInterval),
		redisDB:         utils.MustGetEnvInt(tbaPickCacheRedisDBEnvKey, defaultTbaPickCacheRedisDB),
		redisKeyPrefix:  utils.MustGetEnvString(tbaPickCacheRedisKeyPrefixEnvKey, defaultTbaPickCacheRedisKeyPrefix),
	}
}

func getEventTeamCacheDefaults() *eventTeamCacheDefaults {
	eventTeamDefaultsOnce.Do(func() { eventTeamDefaults = loadEventTeamCacheDefaults() })
	return &eventTeamDefaults
}

// EventTeamCacheConfig groups the configuration values used to create an
// EventTeamCache.
type EventTeamCacheConfig struct {
	Type            string
	RefreshInterval time.Duration
	RedisDB         int
	RedisKeyPrefix  string
}

// EventTeamCacheConfigFromDefaults returns a config populated from environment
// variables.
func EventTeamCacheConfigFromDefaults() EventTeamCacheConfig {
	d := getEventTeamCacheDefaults()
	return EventTeamCacheConfig{
		Type:            strings.ToLower(d.cacheType),
		RefreshInterval: d.refreshInterval,
		RedisDB:         d.redisDB,
		RedisKeyPrefix:  d.redisKeyPrefix,
	}
}

// PickCacheTTL returns the TTL applied to Redis keys backing the pick validation
// cache. It is intentionally longer than the refresh interval so a failed
// refresh does not immediately expire valid data.
func PickCacheTTL() time.Duration {
	return getEventTeamCacheDefaults().refreshInterval * 2
}
