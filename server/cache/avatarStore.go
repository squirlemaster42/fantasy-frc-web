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
)

// TODO I think that there will be too much variance in the avatars requested for this
// to be a reasonable LRU cache and we should always just go to redis.
// Redis should be fast enough anyways since we are loading these after the page loads.
type AvatarStore struct {
	client     *redis.Client
	tbaHandler tbaHandler.TbaHandler
}

func NewAvatarStore(tbaHander tbaHandler.TbaHandler) (AvatarStore, error) {
	// TODO Set options from env file
	// TODO We should not cache the avatars on the default db
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		Protocol: 2,
	})
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

// TODO Figure out if we should allow the user to pass a context
// TODO Maybe we can pass the request context
// TODO Need to figure out the implications of passing different contextes
func (a *AvatarStore) storeAvatar(teamNum int, avatar []byte) error {
	// Store the avatar for 4 weeks
	return a.client.Set(context.Background(), strconv.Itoa(teamNum), avatar, 4*7*24*time.Hour).Err()
}

func (a *AvatarStore) checkCache(teamNum int) ([]byte, error) {
	if a.client == nil {
		return nil, errors.New("Redis not found")
	}

	avatar, err := a.client.Get(context.Background(), strconv.Itoa(teamNum)).Result()
	if err != nil {
		return nil, err
	}
	return []byte(avatar), err
}

func (a *AvatarStore) getTbaAvatar(teamNum int) ([]byte, error) {
	base64Str, err := a.tbaHandler.MakeTeamAvatarRequest(fmt.Sprintf("frc%d", teamNum))
	if err != nil {
		return nil, err
	}

	avatar, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}
	return avatar, nil
}

func (a *AvatarStore) GetAvatar(teamNum int) ([]byte, error) {
	log.DebugNoContext("Loading avatar", "Team Num", teamNum)
	avatar, err := a.checkCache(teamNum)

	if err == redis.Nil {
		log.DebugNoContext("Avatar not in redis, loading from TBA", "Team Num", teamNum)
		avatar, err = a.getTbaAvatar(teamNum)

		err = a.storeAvatar(teamNum, avatar)
		if err != nil {
			log.WarnNoContext("Failed to store avatar in redis", "Error", err)
		}
	} else if err != nil {
		log.WarnNoContext("Failed to get cached avatar", "Team number", teamNum, "Error", err)
		return a.getTbaAvatar(teamNum)
	} else {
		log.DebugNoContext("Avatar in redis", "Team Num", teamNum)
	}

	return avatar, nil
}
