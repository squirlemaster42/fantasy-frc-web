package cache

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"server/log"
	"server/tbaHandler"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/extra/redisotel/v9"
)

// I think that there will be too much variance in the avatars requested for this
// to be a reasonable LRU cache and we should always just go to redis.
// Redis should be fast enough anyways since we are loading these after the page loads.
type AvatarStore struct {
	client     *redis.Client
	tbaHandler tbaHandler.TbaHandler
}

func NewAvatarStore(tbaHander tbaHandler.TbaHandler, redisAddr string, redisPassword string, redisDB int) (AvatarStore, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
		Protocol: 2,
	})
	_ = redisotel.InstrumentTracing(rdb)
	_ = redisotel.InstrumentMetrics(rdb)
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return AvatarStore{
			tbaHandler: tbaHander,
		}, nil
	}
	return AvatarStore{
		client:     rdb,
		tbaHandler: tbaHander,
	}, nil
}

func (a *AvatarStore) Close() error {
	return a.client.Close()
}

func (a *AvatarStore) storeAvatar(ctx context.Context, teamNum int, avatar []byte) error {
	// Store the avatar for 4 weeks
	return a.client.Set(ctx, strconv.Itoa(teamNum), avatar, 4*7*24*time.Hour).Err()
}

func (a *AvatarStore) checkCache(ctx context.Context, teamNum int) ([]byte, error) {
	if a.client == nil {
		return nil, errors.New("Redis not found")
	}

	avatar, err := a.client.Get(ctx, strconv.Itoa(teamNum)).Result()
	if err != nil {
		return nil, err
	}
	return []byte(avatar), err
}

func (a *AvatarStore) getTbaAvatar(ctx context.Context, teamNum int) ([]byte, error) {
	base64Str, err := a.tbaHandler.MakeTeamAvatarRequest(ctx, fmt.Sprintf("frc%d", teamNum))
	if err != nil {
		return nil, err
	}

	avatar, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}
	return avatar, nil
}

func (a *AvatarStore) GetAvatar(ctx context.Context, teamNum int) ([]byte, error) {
	log.DebugNoContext("Loading avatar", "Team Num", teamNum)
	avatar, err := a.checkCache(ctx, teamNum)

	if err == redis.Nil {
		log.DebugNoContext("Avatar not in redis, loading from TBA", "Team Num", teamNum)
		avatar, err = a.getTbaAvatar(ctx, teamNum)
		if err != nil {
			log.Warn(ctx, "Failed to get avatar", "Team Num", teamNum, "Error", err)
			return nil, err
		}

		err = a.storeAvatar(ctx, teamNum, avatar)
		if err != nil {
			log.Warn(ctx, "Failed to store avatar in redis", "Error", err)
		}
	} else if err != nil {
		log.Warn(ctx, "Failed to get cached avatar", "Team number", teamNum, "Error", err)
		return a.getTbaAvatar(ctx, teamNum)
	} else {
		log.DebugNoContext("Avatar in redis", "Team Num", teamNum)
	}

	return avatar, nil
}
