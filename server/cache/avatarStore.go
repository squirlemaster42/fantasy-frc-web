package cache

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"server/log"
	"server/tbaHandler"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type AvatarStore struct {
	client     *redis.Client
	tbaHandler tbaHandler.TbaHandler
}

func NewAvatarStore(tbaHander tbaHandler.TbaHandler) (AvatarStore, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.WarnNoContext("Failed to parse REDIS_URL, using defaults", "URL", redisURL, "Error", err)
		opt = &redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
			Protocol: 2,
		}
	}

	rdb := redis.NewClient(opt)
	ctx := context.Background()
	_, err = rdb.Ping(ctx).Result()
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

// TODO Figure out if we should allow the user to pass a context
// TODO Maybe we can pass the request context
// TODO Need to figure out the implications of passing different contextes
func (a *AvatarStore) storeAvatar(teamNum int, avatar []byte) error {
	// Store the avatar for 4 weeks
	return a.client.Set(context.Background(), strconv.Itoa(teamNum), avatar, 4*7*24*time.Hour).Err()
}

func (a *AvatarStore) checkCache(teamNum int) ([]byte, error) {
	avatar, err := a.client.Get(context.Background(), strconv.Itoa(teamNum)).Result()
	return []byte(avatar), err
}

func (a *AvatarStore) GetAvatar(teamNum int) ([]byte, error) {
	log.DebugNoContext("Loading avatar", "Team Num", teamNum)
	avatar, err := a.checkCache(teamNum)

	if err == redis.Nil {
		log.DebugNoContext("Avatar not in redis, loading from TBA", "Team Num", teamNum)
		base64Str, err := a.tbaHandler.MakeTeamAvatarRequest(fmt.Sprintf("frc%d", teamNum))
		if err != nil {
			return nil, err
		}

		avatar, err = base64.StdEncoding.DecodeString(base64Str)
		if err != nil {
			return nil, err
		}

		err = a.storeAvatar(teamNum, avatar)
		if err != nil {
			log.WarnNoContext("Failed to store avatar in redis", "Error", err)
		}
	} else if err != nil {
		log.WarnNoContext("Failed to get cached avatar", "Team number", teamNum, "Error", err)
		return nil, err
	} else {
		log.DebugNoContext("Avatar in redis", "Team Num", teamNum)
	}

	return avatar, nil
}
