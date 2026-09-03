package cache

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"strconv"

	"github.com/redis/go-redis/v9"

	"server/log"
	"server/tbaHandler"
)

const DefaultAvatarColor = "#2d3136"

type AvatarStoreInterface interface {
	GetAvatar(ctx context.Context, teamNum int) ([]byte, error)
	GetAvatarColor(ctx context.Context, teamNum int) string
	Close() error
}

// I think that there will be too much variance in the avatars requested for this
// to be a reasonable LRU cache and we should always just go to redis.
// Redis should be fast enough anyways since we are loading these after the page loads.
type AvatarStore struct {
	client     *redis.Client
	tbaHandler tbaHandler.TBAInterface
}

func NewAvatarStore(ctx context.Context, tbaHander tbaHandler.TBAInterface, redisAddr string, redisPassword string, redisDB int) (AvatarStore, error) {
		rdb := redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
			Protocol: redisProtocolVersion,
		})
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Error(ctx, "AvatarStore: Redis unavailable, avatar caching disabled", "error", err)
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
	if a.client == nil {
		return nil
	}
	return a.client.Close()
}

func (a *AvatarStore) storeAvatar(ctx context.Context, teamNum int, avatar []byte) error {
	// Store the avatar for the configured TTL.
	return a.client.Set(ctx, strconv.Itoa(teamNum), avatar, AvatarCacheTTL()).Err()
}

func (a *AvatarStore) checkCache(ctx context.Context, teamNum int) ([]byte, error) {
	if a.client == nil {
		return nil, errors.New("redis not found")
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
	log.Debug(ctx, "Loading avatar", "teamNum", teamNum)
	avatar, err := a.checkCache(ctx, teamNum)

	if err == redis.Nil {
		log.Debug(ctx, "Avatar not in redis, loading from TBA", "teamNum", teamNum)
		avatar, err = a.getTbaAvatar(ctx, teamNum)
		if err != nil {
			log.Warn(ctx, "Failed to get avatar", "teamNum", teamNum, "error", err)
			return nil, err
		}

		err = a.storeAvatar(ctx, teamNum, avatar)
		if err != nil {
			log.Warn(ctx, "Failed to store avatar in redis", "error", err)
		}
	} else if err != nil {
		log.Warn(ctx, "Failed to get cached avatar", "teamNum", teamNum, "error", err)
		return a.getTbaAvatar(ctx, teamNum)
	} else {
		log.Debug(ctx, "Avatar in redis", "teamNum", teamNum)
	}

	return avatar, nil
}

// GetAvatarColor returns a CSS hex color that complements the team's avatar.
// The color is cached in Redis with the same TTL as the avatar. If extraction
// fails or Redis is unavailable, it returns the default theme color.
//
// To avoid turning every page view into a burst of TBA requests, this method
// only extracts a color when the avatar bytes are already cached in Redis. If
// the avatar is not cached, it returns the default color and lets the avatar
// image endpoint fetch and cache the avatar on demand.
func (a *AvatarStore) GetAvatarColor(ctx context.Context, teamNum int) string {
	if a.client == nil {
		return DefaultAvatarColor
	}

	colorKey := "avatar:color:" + strconv.Itoa(teamNum)
	cached, err := a.client.Get(ctx, colorKey).Result()
	if err == nil && cached != "" {
		return cached
	}
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Warn(ctx, "Failed to get cached avatar color", "teamNum", teamNum, "error", err)
	}

	avatar, err := a.checkCache(ctx, teamNum)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Warn(ctx, "Failed to check avatar cache for color extraction", "teamNum", teamNum, "error", err)
		}
		return DefaultAvatarColor
	}

	colorStr, err := extractAvatarColor(avatar)
	if err != nil {
		log.Warn(ctx, "Failed to extract avatar color", "teamNum", teamNum, "error", err)
		return DefaultAvatarColor
	}

	if err := a.client.Set(ctx, colorKey, colorStr, AvatarCacheTTL()).Err(); err != nil {
		log.Warn(ctx, "Failed to cache avatar color", "teamNum", teamNum, "error", err)
	}
	return colorStr
}

func extractAvatarColor(avatar []byte) (string, error) {
	img, err := png.Decode(bytes.NewReader(avatar))
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	var rSum, gSum, bSum, count uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A < 128 {
				continue
			}
			// Ignore near-white and near-black pixels so backgrounds and
			// transparency edges don't wash out the team's colors.
			if (c.R > 240 && c.G > 240 && c.B > 240) || (c.R < 15 && c.G < 15 && c.B < 15) {
				continue
			}
			rSum += uint64(c.R)
			gSum += uint64(c.G)
			bSum += uint64(c.B)
			count++
		}
	}
	if count == 0 {
		return "", errors.New("no usable pixels for color extraction")
	}

	avgR := uint8(rSum / count)
	avgG := uint8(gSum / count)
	avgB := uint8(bSum / count)
	return backgroundColorFromAvatarColor(avgR, avgG, avgB), nil
}

func backgroundColorFromAvatarColor(r, g, b uint8) string {
	h, s, l := rgbToHsl(r, g, b)
	if s > 0.7 {
		s = 0.7
	}
	if l > 0.5 {
		l = 0.18
	} else {
		l = 0.32
	}
	nr, ng, nb := hslToRgb(h, s, l)
	return fmt.Sprintf("#%02x%02x%02x", nr, ng, nb)
}

func rgbToHsl(r, g, b uint8) (h, s, l float64) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	l = (max + min) / 2.0

	if max == min {
		return 0, 0, l
	}

	d := max - min
	if l > 0.5 {
		s = d / (2.0 - max - min)
	} else {
		s = d / (max + min)
	}

	switch max {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6.0
		}
	case gf:
		h = (bf-rf)/d + 2.0
	case bf:
		h = (rf-gf)/d + 4.0
	}
	h /= 6.0
	return h, s, l
}

func hslToRgb(h, s, l float64) (r, g, b uint8) {
	if s == 0 {
		v := uint8(l*255.0 + 0.5)
		return v, v, v
	}

	var q float64
	if l < 0.5 {
		q = l * (1.0 + s)
	} else {
		q = l + s - l*s
	}
	p := 2.0*l - q

	r = uint8(hueToRgb(p, q, h+1.0/3.0)*255.0 + 0.5)
	g = uint8(hueToRgb(p, q, h)*255.0 + 0.5)
	b = uint8(hueToRgb(p, q, h-1.0/3.0)*255.0 + 0.5)
	return r, g, b
}

func hueToRgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1.0
	}
	if t > 1 {
		t -= 1.0
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6.0*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6.0
	}
	return p
}
