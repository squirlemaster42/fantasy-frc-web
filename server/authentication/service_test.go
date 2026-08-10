package authentication

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"server/model/mocks"
)

func newTestAuthService(t *testing.T) (*authService, *mocks.MockUserStore, PasswordHasher) {
	hasher, err := NewBcryptPasswordHasher(bcrypt.MinCost)
	require.NoError(t, err)

	userStore := mocks.NewMockUserStore(t)
	service := NewAuthService(userStore, hasher, AuthConfig{
		MinPasswordLength:           8,
		MinUsernameLength:           3,
		MaxUsernameLength:           32,
		UsernameAllowedSpecialChars: "_-",
	}).(*authService)

	return service, userStore, hasher
}

func TestAuthService_Login_Success(t *testing.T) {
	service, userStore, hasher := newTestAuthService(t)
	ctx := context.Background()

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	passwordHash, err := hasher.Hash("Secret123!")
	require.NoError(t, err)

	userStore.On("GetPasswordHashByUsername", ctx, "testuser").Return(passwordHash, nil)
	userStore.On("GetUserUuidByUsername", ctx, "testuser").Return(userUuid, nil)
	userStore.On("RegisterSession", ctx, userUuid, mock.AnythingOfType("string")).Return(nil)

	returnedUuid, sessionToken, err := service.Login(ctx, "testuser", "Secret123!")
	require.NoError(t, err)
	assert.Equal(t, userUuid, returnedUuid)
	assert.NotEmpty(t, sessionToken)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	service, userStore, hasher := newTestAuthService(t)
	ctx := context.Background()

	passwordHash, err := hasher.Hash("Secret123!")
	require.NoError(t, err)

	userStore.On("GetPasswordHashByUsername", ctx, "testuser").Return(passwordHash, nil)

	_, _, err = service.Login(ctx, "testuser", "wrongpassword")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_UnknownUsername(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	userStore.On("GetPasswordHashByUsername", ctx, "unknown").Return("", sql.ErrNoRows)

	_, _, err := service.Login(ctx, "unknown", "anypassword")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Register_Success(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	userStore.On("UsernameTaken", ctx, "newuser").Return(false, nil)
	userStore.On("RegisterUser", ctx, "newuser", mock.AnythingOfType("string")).Return(userUuid, nil)
	userStore.On("RegisterSession", ctx, userUuid, mock.AnythingOfType("string")).Return(nil)

	returnedUuid, sessionToken, err := service.Register(ctx, "newuser", "Secret123!")
	require.NoError(t, err)
	assert.Equal(t, userUuid, returnedUuid)
	assert.NotEmpty(t, sessionToken)
}

func TestAuthService_Register_UsernameTaken(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	userStore.On("UsernameTaken", ctx, "existing").Return(true, nil)

	_, _, err := service.Register(ctx, "existing", "Secret123!")
	assert.ErrorIs(t, err, ErrUsernameTaken)
}

func TestAuthService_Register_InvalidUsername(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	_, _, err := service.Register(ctx, "ab", "Secret123!")
	assert.True(t, IsValidationError(err))
	userStore.AssertExpectations(t)
}

func TestAuthService_Register_WeakPassword(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	_, _, err := service.Register(ctx, "newuser", "short")
	assert.True(t, IsValidationError(err))
	userStore.AssertExpectations(t)
}

func TestAuthService_Logout(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	userStore.On("UnRegisterSession", ctx, "session-token").Return(nil)

	assert.NoError(t, service.Logout(ctx, "session-token"))
}

func TestAuthService_ChangePassword_Success(t *testing.T) {
	service, userStore, hasher := newTestAuthService(t)
	ctx := context.Background()

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	passwordHash, err := hasher.Hash("OldPass123!")
	require.NoError(t, err)

	userStore.On("GetPasswordHashByUsername", ctx, "testuser").Return(passwordHash, nil)
	userStore.On("UpdatePassword", ctx, "testuser", mock.AnythingOfType("string")).Return(nil)
	userStore.On("InvalidateAllUserSessionsExcept", ctx, userUuid, "").Return(nil)

	assert.NoError(t, service.ChangePassword(ctx, userUuid, "testuser", "OldPass123!", "NewPass456!"))
}

func TestAuthService_ChangePassword_InvalidCurrentPassword(t *testing.T) {
	service, userStore, hasher := newTestAuthService(t)
	ctx := context.Background()

	passwordHash, err := hasher.Hash("OldPass123!")
	require.NoError(t, err)

	userStore.On("GetPasswordHashByUsername", ctx, "testuser").Return(passwordHash, nil)

	err = service.ChangePassword(ctx, uuid.UUID{}, "testuser", "wrong", "NewPass456!")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_ChangePassword_WeakNewPassword(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	err := service.ChangePassword(ctx, uuid.UUID{}, "testuser", "OldPass123!", "short")
	assert.True(t, IsValidationError(err))
	userStore.AssertExpectations(t)
}

func TestAuthService_ValidateSession_Success(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	userStore.On("ValidateSessionToken", ctx, "valid-token").Return(true, nil)
	userStore.On("GetUserBySessionToken", ctx, "valid-token").Return(userUuid, nil)

	returnedUuid, err := service.ValidateSession(ctx, "valid-token")
	require.NoError(t, err)
	assert.Equal(t, userUuid, returnedUuid)
}

func TestAuthService_ValidateSession_Invalid(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	userStore.On("ValidateSessionToken", ctx, "invalid-token").Return(false, nil)

	_, err := service.ValidateSession(ctx, "invalid-token")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_InvalidateOtherSessions(t *testing.T) {
	service, userStore, _ := newTestAuthService(t)
	ctx := context.Background()

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	userStore.On("InvalidateAllUserSessionsExcept", ctx, userUuid, "keep-token").Return(nil)

	assert.NoError(t, service.InvalidateOtherSessions(ctx, userUuid, "keep-token"))
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Message: "something failed"}
	assert.Equal(t, "something failed", err.Error())
	assert.True(t, IsValidationError(err))
	assert.False(t, IsValidationError(errors.New("plain error")))
}
