package handler

import (
	"context"

	"server/cache"
)

type mockAvatarStore struct {
	avatar []byte
	color  string
	err    error
}

func (m *mockAvatarStore) GetAvatar(ctx context.Context, teamNum int) ([]byte, error) {
	return m.avatar, m.err
}

func (m *mockAvatarStore) GetAvatarColor(ctx context.Context, teamNum int) string {
	if m.color == "" {
		return cache.DefaultAvatarColor
	}
	return m.color
}

func (m *mockAvatarStore) Close() error {
	return nil
}
