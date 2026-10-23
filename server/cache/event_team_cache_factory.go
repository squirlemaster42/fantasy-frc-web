package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"server/log"
	"server/tbaHandler"
)

// NewEventTeamCache creates an EventTeamCache using the configured backend. If
// the configured backend is Redis and Redis is unreachable, it falls back to an
// in-memory cache so that pick validation can still function.
func NewEventTeamCache(ctx context.Context, handler tbaHandler.TBAInterface, redisAddr string, redisPassword string, config EventTeamCacheConfig) (EventTeamCache, error) {
	switch config.Type {
	case "redis":
		client := redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       config.RedisDB,
			Protocol: redisProtocolVersion,
		})
		_, err := client.Ping(ctx).Result()
		if err != nil {
			log.Warn(ctx, "Redis pick validation cache unavailable, falling back to memory cache", "error", err, "redisAddr", redisAddr, "redisDB", config.RedisDB)
			return newMemoryEventTeamCache(handler), nil
		}
		return newRedisEventTeamCache(handler, client, config.RedisKeyPrefix+"teams"), nil
	case "memory":
		return newMemoryEventTeamCache(handler), nil
	default:
		log.Warn(ctx, "Unknown pick validation cache type, using memory cache", "type", config.Type)
		return newMemoryEventTeamCache(handler), nil
	}
}

// StartEventTeamCacheRefreshLoop begins a background goroutine that refreshes
// the cache immediately and then on every configured interval. The loop exits
// when the provided context is cancelled.
func StartEventTeamCacheRefreshLoop(ctx context.Context, cache EventTeamCache, interval time.Duration) {
	go func() {
		if err := cache.Refresh(ctx); err != nil {
			log.Error(ctx, "Failed initial pick validation cache refresh", "error", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := cache.Refresh(ctx); err != nil {
					log.Error(ctx, "Failed to refresh pick validation cache", "error", err)
				}
			}
		}
	}()
}

// CloseEventTeamCache closes any underlying resources (e.g., Redis clients)
// held by the cache.
func CloseEventTeamCache(cache EventTeamCache) error {
	if cache == nil {
		return nil
	}
	if redisCache, ok := cache.(*redisEventTeamCache); ok {
		if err := redisCache.client.Close(); err != nil {
			return fmt.Errorf("failed to close event team redis client: %w", err)
		}
	}
	return nil
}
