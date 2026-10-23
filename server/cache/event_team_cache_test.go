package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"server/swagger"
)

type testTBAHandlerForCache struct {
	teams map[string][]swagger.Team
	err   error
}

func (t *testTBAHandlerForCache) MakeEventListReq(ctx context.Context, teamId string) ([]string, error) {
	return nil, nil
}

func (t *testTBAHandlerForCache) MakeMatchReq(ctx context.Context, matchId string) (swagger.Match, error) {
	return swagger.Match{}, nil
}

func (t *testTBAHandlerForCache) MakeEventMatchKeysRequest(ctx context.Context, eventId string) ([]string, error) {
	return nil, nil
}

func (t *testTBAHandlerForCache) MakeTeamsAtEventRequest(ctx context.Context, eventId string) ([]swagger.Team, error) {
	if t.err != nil {
		return nil, t.err
	}
	return t.teams[eventId], nil
}

func (t *testTBAHandlerForCache) MakeEliminationAllianceRequest(ctx context.Context, eventId string) ([]swagger.EliminationAlliance, error) {
	return nil, nil
}

func (t *testTBAHandlerForCache) MakeTeamAvatarRequest(ctx context.Context, teamId string) (string, error) {
	return "", nil
}

func newTestTBAHandler() *testTBAHandlerForCache {
	return &testTBAHandlerForCache{
		teams: map[string][]swagger.Team{
			"2026arc": {
				{Key: "frc254"},
				{Key: "frc1114"},
			},
			"2026cur": {
				{Key: "frc254"},
				{Key: "frc1678"},
			},
		},
	}
}

func TestMemoryEventTeamCache_RefreshAndLookup(t *testing.T) {
	handler := newTestTBAHandler()
	cache := newMemoryEventTeamCache(handler)

	ctx := t.Context()
	err := cache.Refresh(ctx)
	require.NoError(t, err)

	valid, err := cache.IsTeamAtEvent(ctx, "frc254")
	require.NoError(t, err)
	assert.True(t, valid)

	valid, err = cache.IsTeamAtEvent(ctx, "frc1678")
	require.NoError(t, err)
	assert.True(t, valid)

	valid, err = cache.IsTeamAtEvent(ctx, "frc9999")
	require.NoError(t, err)
	assert.False(t, valid)

	assert.False(t, cache.LastRefresh().IsZero())
}

func TestMemoryEventTeamCache_NotReady(t *testing.T) {
	cache := newMemoryEventTeamCache(newTestTBAHandler())

	valid, err := cache.IsTeamAtEvent(t.Context(), "frc254")
	require.Error(t, err)
	assert.Equal(t, ErrCacheNotReady, err)
	assert.False(t, valid)
}

func TestMemoryEventTeamCache_RefreshError(t *testing.T) {
	handler := newTestTBAHandler()
	handler.err = errors.New("tba unavailable")
	cache := newMemoryEventTeamCache(handler)

	err := cache.Refresh(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tba unavailable")
}

func TestMemoryEventTeamCache_InvalidateAndRefresh(t *testing.T) {
	handler := newTestTBAHandler()
	cache := newMemoryEventTeamCache(handler)

	ctx := t.Context()
	require.NoError(t, cache.Refresh(ctx))
	require.NoError(t, cache.InvalidateAndRefresh(ctx))

	valid, err := cache.IsTeamAtEvent(ctx, "frc254")
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestRedisEventTeamCache_RefreshAndLookup(t *testing.T) {
	mr := miniredis.RunT(t)
	handler := newTestTBAHandler()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	cache := newRedisEventTeamCache(handler, client, "test:pick_validation:")

	ctx := t.Context()
	err := cache.Refresh(ctx)
	require.NoError(t, err)

	valid, err := cache.IsTeamAtEvent(ctx, "frc254")
	require.NoError(t, err)
	assert.True(t, valid)

	valid, err = cache.IsTeamAtEvent(ctx, "frc9999")
	require.NoError(t, err)
	assert.False(t, valid)

	assert.False(t, cache.LastRefresh().IsZero())
}

func TestRedisEventTeamCache_NotReady(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	cache := newRedisEventTeamCache(newTestTBAHandler(), client, "test:pick_validation:")

	valid, err := cache.IsTeamAtEvent(t.Context(), "frc254")
	require.Error(t, err)
	assert.Equal(t, ErrCacheNotReady, err)
	assert.False(t, valid)
}

func TestRedisEventTeamCache_InvalidateAndRefresh(t *testing.T) {
	mr := miniredis.RunT(t)
	handler := newTestTBAHandler()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	cache := newRedisEventTeamCache(handler, client, "test:pick_validation:")

	ctx := t.Context()
	require.NoError(t, cache.Refresh(ctx))
	require.NoError(t, cache.InvalidateAndRefresh(ctx))

	valid, err := cache.IsTeamAtEvent(ctx, "frc254")
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestNewEventTeamCache_MemoryBackend(t *testing.T) {
	config := EventTeamCacheConfig{Type: "memory"}
	cache, err := NewEventTeamCache(t.Context(), newTestTBAHandler(), "", "", config)
	require.NoError(t, err)
	assert.NotNil(t, cache)

	_, ok := cache.(*memoryEventTeamCache)
	assert.True(t, ok)
}

func TestNewEventTeamCache_RedisBackend(t *testing.T) {
	mr := miniredis.RunT(t)
	config := EventTeamCacheConfig{Type: "redis", RedisDB: 0, RedisKeyPrefix: "test:"}
	cache, err := NewEventTeamCache(t.Context(), newTestTBAHandler(), mr.Addr(), "", config)
	require.NoError(t, err)
	assert.NotNil(t, cache)

	_, ok := cache.(*redisEventTeamCache)
	assert.True(t, ok)
}

func TestNewEventTeamCache_RedisUnavailableFallsBackToMemory(t *testing.T) {
	config := EventTeamCacheConfig{Type: "redis", RedisDB: 0, RedisKeyPrefix: "test:"}
	cache, err := NewEventTeamCache(t.Context(), newTestTBAHandler(), "127.0.0.1:1", "", config)
	require.NoError(t, err)
	assert.NotNil(t, cache)

	_, ok := cache.(*memoryEventTeamCache)
	assert.True(t, ok)
}

func TestNewEventTeamCache_UnknownBackendDefaultsToMemory(t *testing.T) {
	config := EventTeamCacheConfig{Type: "unknown"}
	cache, err := NewEventTeamCache(t.Context(), newTestTBAHandler(), "", "", config)
	require.NoError(t, err)
	assert.NotNil(t, cache)

	_, ok := cache.(*memoryEventTeamCache)
	assert.True(t, ok)
}

func TestPickCacheTTL(t *testing.T) {
	// Default refresh interval is 1h, so TTL should be 2h.
	ttl := PickCacheTTL()
	assert.Equal(t, 2*time.Hour, ttl)
}
