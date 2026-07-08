package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"server/tbaHandler"
)

func TestNewAvatarStore_WithRedis(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), *tbaHandler, s.Addr(), "", 0)

	assert.NoError(t, err)
	assert.NotNil(t, store.client)
	assert.NoError(t, store.Close())
}

func TestNewAvatarStore_WithoutRedis(t *testing.T) {
	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), *tbaHandler, "localhost:1", "", 0)

	assert.NoError(t, err)
	assert.Nil(t, store.client)
}

func TestAvatarStore_Close_WithNilClient(t *testing.T) {
	store := AvatarStore{}
	assert.NoError(t, store.Close())
}

func TestAvatarStore_storeAvatarAndCheckCache(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), *tbaHandler, s.Addr(), "", 0)
	assert.NoError(t, err)
	defer func() { _ = store.Close() }()

	avatar := []byte("fake-avatar-bytes")
	err = store.storeAvatar(context.Background(), 254, avatar)
	assert.NoError(t, err)

	cached, err := store.checkCache(context.Background(), 254)
	assert.NoError(t, err)
	assert.Equal(t, avatar, cached)
}

func TestAvatarStore_checkCache_Miss(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), *tbaHandler, s.Addr(), "", 0)
	assert.NoError(t, err)
	defer func() { _ = store.Close() }()

	cached, err := store.checkCache(context.Background(), 254)
	assert.Equal(t, redis.Nil, err)
	assert.Nil(t, cached)
}

func TestAvatarStore_checkCache_NoRedis(t *testing.T) {
	store := AvatarStore{}

	cached, err := store.checkCache(context.Background(), 254)
	assert.Error(t, err)
	assert.Nil(t, cached)
}

func TestAvatarStore_GetAvatar_CacheHit(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), *tbaHandler, s.Addr(), "", 0)
	assert.NoError(t, err)
	defer func() { _ = store.Close() }()

	// storeAvatar stores raw bytes in Redis, not base64-encoded bytes
	avatar := []byte("fake-avatar-bytes")
	err = s.Set("254", string(avatar))
	assert.NoError(t, err)

	result, err := store.GetAvatar(context.Background(), 254)
	assert.NoError(t, err)
	assert.Equal(t, avatar, result)
}

func TestAvatarStore_GetAvatar_NoRedis(t *testing.T) {
	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), *tbaHandler, "localhost:1", "", 0)
	assert.NoError(t, err)

	// Without Redis, the store falls back to the TBA handler, which will fail
	// because there is no real TBA API available in this test.
	result, err := store.GetAvatar(context.Background(), 254)
	assert.Error(t, err)
	assert.Nil(t, result)
}
