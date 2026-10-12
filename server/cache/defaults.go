package cache

import (
	"server/utils"
	"sync"
	"time"
)

const (
	avatarCacheTTLEnvKey = "AVATAR_CACHE_TTL"
	defaultAvatarCacheTTL = 28 * 24 * time.Hour

	redisProtocolVersion = 2
)

var (
	defaultsOnce sync.Once
	defaults     cacheDefaults
)

type cacheDefaults struct {
	avatarCacheTTL time.Duration
}

func loadDefaults() cacheDefaults {
	return cacheDefaults{
		avatarCacheTTL: utils.MustGetEnvDuration(avatarCacheTTLEnvKey, defaultAvatarCacheTTL),
	}
}

func getDefaults() *cacheDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// AvatarCacheTTL returns how long avatars are cached in Redis.
func AvatarCacheTTL() time.Duration { return getDefaults().avatarCacheTTL }
