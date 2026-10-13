package authentication

import (
	"context"

	"uuid"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a testify-based mock implementation of AuthService for tests.
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(ctx context.Context, username, password string) (uuid.UUID, string, error) {
	args := m.Called(ctx, username, password)
	return args.Get(0).(uuid.UUID), args.String(1), args.Error(2)
}

func (m *MockAuthService) Register(ctx context.Context, username, password, confirmPassword string) (uuid.UUID, string, error) {
	args := m.Called(ctx, username, password, confirmPassword)
	return args.Get(0).(uuid.UUID), args.String(1), args.Error(2)
}

func (m *MockAuthService) Logout(ctx context.Context, sessionToken string) error {
	return m.Called(ctx, sessionToken).Error(0)
}

func (m *MockAuthService) ChangePassword(ctx context.Context, userUuid uuid.UUID, username, currentPassword, newPassword string) error {
	return m.Called(ctx, userUuid, username, currentPassword, newPassword).Error(0)
}

func (m *MockAuthService) ValidateSession(ctx context.Context, sessionToken string) (uuid.UUID, error) {
	args := m.Called(ctx, sessionToken)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockAuthService) InvalidateOtherSessions(ctx context.Context, userUuid uuid.UUID, keepSessionToken string) error {
	return m.Called(ctx, userUuid, keepSessionToken).Error(0)
}

// Ensure MockAuthService implements AuthService.
var _ AuthService = (*MockAuthService)(nil)
