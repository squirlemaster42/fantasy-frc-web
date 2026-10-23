package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"server/log"
	"server/tbaHandler"
	"server/utils"
)

// redisEventTeamCache is a Redis-backed implementation of EventTeamCache.
type redisEventTeamCache struct {
	client  *redis.Client
	handler tbaHandler.TBAInterface
	events  []string
	key     string
}

func newRedisEventTeamCache(handler tbaHandler.TBAInterface, client *redis.Client, key string) *redisEventTeamCache {
	return &redisEventTeamCache{
		client:  client,
		handler: handler,
		events:  utils.Events(),
		key:     key,
	}
}

func (r *redisEventTeamCache) Refresh(ctx context.Context) error {
	teams, err := fetchEventTeams(ctx, r.handler, r.events)
	if err != nil {
		return err
	}

	pipe := r.client.Pipeline()
	pipe.Del(ctx, r.key)
	if len(teams) > 0 {
		pipe.SAdd(ctx, r.key, teamSlice(teams)...)
	}
	pipe.Expire(ctx, r.key, PickCacheTTL())
	pipe.Set(ctx, r.lastRefreshKey(), strconv.FormatInt(time.Now().UTC().Unix(), 10), PickCacheTTL())
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute redis pipeline for pick validation cache: %w", err)
	}

	log.Info(ctx, "Refreshed Redis pick validation cache", "events", len(r.events), "teams", len(teams))
	return nil
}

func (r *redisEventTeamCache) InvalidateAndRefresh(ctx context.Context) error {
	err := r.client.Del(ctx, r.key, r.lastRefreshKey()).Err()
	if err != nil {
		log.Warn(ctx, "Failed to invalidate Redis pick validation cache keys", "error", err)
	}
	return r.Refresh(ctx)
}

func (r *redisEventTeamCache) IsTeamAtEvent(ctx context.Context, teamId string) (bool, error) {
	exists, err := r.client.Exists(ctx, r.key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check pick validation cache existence in redis: %w", err)
	}
	if exists == 0 {
		return false, ErrCacheNotReady
	}

	member, err := r.client.SIsMember(ctx, r.key, teamId).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check team membership in redis pick validation cache: %w", err)
	}
	return member, nil
}

func (r *redisEventTeamCache) LastRefresh() time.Time {
	ctx := context.Background()
	val, err := r.client.Get(ctx, r.lastRefreshKey()).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Warn(ctx, "Failed to read Redis pick validation cache refresh time", "error", err)
		}
		return time.Time{}
	}

	unix, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		log.Warn(ctx, "Failed to parse Redis pick validation cache refresh time", "error", err)
		return time.Time{}
	}
	return time.Unix(unix, 0).UTC()
}

func (r *redisEventTeamCache) lastRefreshKey() string {
	return r.key + ":last_refresh"
}
