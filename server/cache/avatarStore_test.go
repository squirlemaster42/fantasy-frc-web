package cache

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"server/swagger"
	"server/tbaHandler"
)

func TestNewAvatarStore_WithRedis(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), tbaHandler, s.Addr(), "", 0)

	assert.NoError(t, err)
	assert.NotNil(t, store.client)
	assert.NoError(t, store.Close())
}

func TestNewAvatarStore_WithoutRedis(t *testing.T) {
	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), tbaHandler, "localhost:1", "", 0)

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
	store, err := NewAvatarStore(context.Background(), tbaHandler, s.Addr(), "", 0)
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
	store, err := NewAvatarStore(context.Background(), tbaHandler, s.Addr(), "", 0)
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
	store, err := NewAvatarStore(context.Background(), tbaHandler, s.Addr(), "", 0)
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
	store, err := NewAvatarStore(context.Background(), tbaHandler, "localhost:1", "", 0)
	assert.NoError(t, err)

	// Without Redis, the store falls back to the TBA handler, which will fail
	// because there is no real TBA API available in this test.
	result, err := store.GetAvatar(context.Background(), 254)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// mockTBAHandler is a test double that returns a configurable base64 avatar
// or error from MakeTeamAvatarRequest.
type mockTBAHandler struct {
	avatar string
	err    error
}

func (m *mockTBAHandler) MakeEventListReq(context.Context, string) ([]string, error) {
	return nil, nil
}

func (m *mockTBAHandler) MakeMatchReq(context.Context, string) (swagger.Match, error) {
	return swagger.Match{}, nil
}

func (m *mockTBAHandler) MakeEventMatchKeysRequest(context.Context, string) ([]string, error) {
	return nil, nil
}

func (m *mockTBAHandler) MakeTeamsAtEventRequest(context.Context, string) ([]swagger.Team, error) {
	return nil, nil
}

func (m *mockTBAHandler) MakeEliminationAllianceRequest(context.Context, string) ([]swagger.EliminationAlliance, error) {
	return nil, nil
}

func (m *mockTBAHandler) MakeTeamAvatarRequest(context.Context, string) (string, error) {
	return m.avatar, m.err
}

func TestAvatarStore_GetAvatar_RedisErrorFallsBackToTBA(t *testing.T) {
	s := miniredis.RunT(t)

	expectedAvatar := []byte("avatar")
	mock := &mockTBAHandler{
		avatar: base64.StdEncoding.EncodeToString(expectedAvatar),
	}
	store, err := NewAvatarStore(context.Background(), mock, s.Addr(), "", 0)
	assert.NoError(t, err)
	defer func() { _ = store.Close() }()

	// Force Redis connection errors after the store has been initialized.
	s.Close()

	result, err := store.GetAvatar(context.Background(), 254)
	assert.NoError(t, err)
	assert.Equal(t, expectedAvatar, result)
}

// setFailureHook injects a failure for Redis SET commands while leaving GET
// behavior untouched, allowing us to exercise the cache-miss/store-failure path.
type setFailureHook struct{}

func (h *setFailureHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *setFailureHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() == "set" {
			cmd.SetErr(errors.New("redis set failed"))
		}
		return err
	}
}

func (h *setFailureHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		err := next(ctx, cmds)
		for _, cmd := range cmds {
			if cmd.Name() == "set" {
				cmd.SetErr(errors.New("redis set failed"))
			}
		}
		return err
	}
}

func TestAvatarStore_GetAvatar_CacheMissSetFailureReturnsAvatar(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	expectedAvatar := []byte("avatar")
	mock := &mockTBAHandler{
		avatar: base64.StdEncoding.EncodeToString(expectedAvatar),
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     s.Addr(),
		Protocol: redisProtocolVersion,
	})
	rdb.AddHook(&setFailureHook{})
	defer func() { _ = rdb.Close() }()

	store := AvatarStore{
		client:     rdb,
		tbaHandler: mock,
	}

	result, err := store.GetAvatar(context.Background(), 254)
	assert.NoError(t, err)
	assert.Equal(t, expectedAvatar, result)
}

func TestAvatarStore_GetAvatar_RedisDownAndTbaDown(t *testing.T) {
	s := miniredis.RunT(t)

	mock := &mockTBAHandler{
		err: errors.New("tba unavailable"),
	}
	store, err := NewAvatarStore(context.Background(), mock, s.Addr(), "", 0)
	assert.NoError(t, err)
	defer func() { _ = store.Close() }()

	// Force Redis connection errors after the store has been initialized.
	s.Close()

	result, err := store.GetAvatar(context.Background(), 254)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func createTestAvatar(t *testing.T, c color.Color) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, c)
		}
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	assert.NoError(t, err)
	return buf.Bytes()
}

func TestExtractAvatarColor(t *testing.T) {
	redAvatar := createTestAvatar(t, color.NRGBA{R: 255, G: 0, B: 0, A: 255})

	colorStr, err := extractAvatarColor(redAvatar)
	assert.NoError(t, err)
	assert.NotEqual(t, DefaultAvatarColor, colorStr)
	assert.Contains(t, colorStr, "#")
}

func TestExtractAvatarColor_NoUsablePixels(t *testing.T) {
	transparentAvatar := createTestAvatar(t, color.NRGBA{R: 255, G: 0, B: 0, A: 0})

	colorStr, err := extractAvatarColor(transparentAvatar)
	assert.Error(t, err)
	assert.Equal(t, "", colorStr)
}

func TestAvatarStore_GetAvatarColor_NoRedis(t *testing.T) {
	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), tbaHandler, "localhost:1", "", 0)
	assert.NoError(t, err)

	color := store.GetAvatarColor(context.Background(), 254)
	assert.Equal(t, DefaultAvatarColor, color)
}

func TestAvatarStore_GetAvatarColor_CacheHit(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), tbaHandler, s.Addr(), "", 0)
	assert.NoError(t, err)
	defer func() { _ = store.Close() }()

	err = s.Set("avatar:color:254", "#abcdef")
	assert.NoError(t, err)

	color := store.GetAvatarColor(context.Background(), 254)
	assert.Equal(t, "#abcdef", color)
}

func TestAvatarStore_GetAvatarColor_AvatarCacheMiss(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), tbaHandler, s.Addr(), "", 0)
	assert.NoError(t, err)
	defer func() { _ = store.Close() }()

	// No avatar cached, so the color should fall back to the default without
	// hitting TBA.
	color := store.GetAvatarColor(context.Background(), 254)
	assert.Equal(t, DefaultAvatarColor, color)
}

func TestAvatarStore_GetAvatarColor_AvatarCacheHit(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	avatar := createTestAvatar(t, color.NRGBA{R: 0, G: 128, B: 255, A: 255})
	tbaHandler := tbaHandler.NewHandler("", nil)
	store, err := NewAvatarStore(context.Background(), tbaHandler, s.Addr(), "", 0)
	assert.NoError(t, err)
	defer func() { _ = store.Close() }()

	err = s.Set("254", string(avatar))
	assert.NoError(t, err)

	color := store.GetAvatarColor(context.Background(), 254)
	assert.NotEqual(t, DefaultAvatarColor, color)
	assert.Contains(t, color, "#")

	cached, err := s.Get("avatar:color:254")
	assert.NoError(t, err)
	assert.Equal(t, color, cached)
}
