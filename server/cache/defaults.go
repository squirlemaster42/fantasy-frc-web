package cache

import (
	"sync"
	"time"

	"server/config"
)

var (
	defaultsOnce sync.Once
	defaults     cacheDefaults
)

type cacheDefaults struct {
	AvatarCacheTTL time.Duration `env:"AVATAR_CACHE_TTL" default:"672h"`
}

func loadDefaults() cacheDefaults {
	var d cacheDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *cacheDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// AvatarCacheTTL returns how long avatars are cached in Redis.
func AvatarCacheTTL() time.Duration { return getDefaults().AvatarCacheTTL }
