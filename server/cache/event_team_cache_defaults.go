package cache

import (
	"strings"
	"sync"
	"time"

	"server/config"
)

var (
	eventTeamDefaultsOnce sync.Once
	eventTeamDefaults     eventTeamCacheDefaults
)

type eventTeamCacheDefaults struct {
	CacheType       string        `env:"TBA_PICK_CACHE_TYPE" default:"memory"`
	RefreshInterval time.Duration `env:"TBA_PICK_CACHE_REFRESH_INTERVAL" default:"1h"`
	RedisDB         int           `env:"TBA_PICK_CACHE_REDIS_DB" default:"3"`
	RedisKeyPrefix  string        `env:"TBA_PICK_CACHE_REDIS_KEY_PREFIX" default:"tba:pick_validation:"`
}

func loadEventTeamCacheDefaults() eventTeamCacheDefaults {
	var d eventTeamCacheDefaults
	config.MustLoad(&d)
	return d
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
		Type:            strings.ToLower(d.CacheType),
		RefreshInterval: d.RefreshInterval,
		RedisDB:         d.RedisDB,
		RedisKeyPrefix:  d.RedisKeyPrefix,
	}
}

// PickCacheTTL returns the TTL applied to Redis keys backing the pick validation
// cache. It is intentionally longer than the refresh interval so a failed
// refresh does not immediately expire valid data.
func PickCacheTTL() time.Duration {
	return getEventTeamCacheDefaults().RefreshInterval * 2
}
