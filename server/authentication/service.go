package authentication

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/log"
	"server/model"

	"github.com/google/uuid"
)

// AuthConfig holds validation settings for authentication flows.
type AuthConfig struct {
	MinPasswordLength           int
	MinUsernameLength           int
	MaxUsernameLength           int
	UsernameAllowedSpecialChars string
}

// AuthService orchestrates authentication flows such as login, registration,
// password changes, and session validation.
type AuthService interface {
	// Login validates credentials and returns the user UUID and a new session token.
	Login(ctx context.Context, username, password string) (uuid.UUID, string, error)
	// Register creates a new user and returns the user UUID and a new session token.
	Register(ctx context.Context, username, password string) (uuid.UUID, string, error)
	// Logout invalidates the given session token.
	Logout(ctx context.Context, sessionToken string) error
	// ChangePassword validates the current password and updates to the new password.
	ChangePassword(ctx context.Context, userUuid uuid.UUID, username, currentPassword, newPassword string) error
	// ValidateSession returns the user UUID for a valid session token.
	ValidateSession(ctx context.Context, sessionToken string) (uuid.UUID, error)
	// InvalidateOtherSessions invalidates all sessions for the user except the given token.
	InvalidateOtherSessions(ctx context.Context, userUuid uuid.UUID, keepSessionToken string) error
}

type authService struct {
	userStore      model.UserStore
	passwordHasher PasswordHasher
	config         AuthConfig
}

// NewAuthService creates an AuthService backed by the given user store and password hasher.
func NewAuthService(userStore model.UserStore, passwordHasher PasswordHasher, config AuthConfig) AuthService {
	return &authService{
		userStore:      userStore,
		passwordHasher: passwordHasher,
		config:         config,
	}
}

func (s *authService) Login(ctx context.Context, username, password string) (uuid.UUID, string, error) {
	var nilUuid uuid.UUID

	username, errMsg := validateUsername(username, s.config.MinUsernameLength, s.config.MaxUsernameLength, s.config.UsernameAllowedSpecialChars)
	if errMsg != "" {
		log.Warn(ctx, "Login username validation failed", "username", username)
		return nilUuid, "", ErrInvalidCredentials
	}

	passwordHash, err := s.userStore.GetPasswordHashByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Constant-time dummy comparison to prevent username enumeration
			_ = s.passwordHasher.Compare(password, string(s.passwordHasher.DummyHash()))
			return nilUuid, "", ErrInvalidCredentials
		}
		log.Error(ctx, "Failed to get password hash", "username", username, "error", err)
		return nilUuid, "", fmt.Errorf("failed to validate login: %w", err)
	}

	if err := s.passwordHasher.Compare(password, passwordHash); err != nil {
		log.Warn(ctx, "Failed login attempt", "username", username)
		return nilUuid, "", ErrInvalidCredentials
	}

	log.Info(ctx, "Valid login attempt for user", "username", username)
	userUuid, err := s.userStore.GetUserUuidByUsername(ctx, username)
	if err != nil {
		log.Error(ctx, "Failed to get user uuid", "username", username, "error", err)
		return nilUuid, "", fmt.Errorf("failed to validate login: %w", err)
	}

	sessionToken, err := generateSessionToken()
	if err != nil {
		log.Error(ctx, "Failed to generate session token", "error", err)
		return nilUuid, "", fmt.Errorf("failed to create session: %w", err)
	}

	if err := s.userStore.RegisterSession(ctx, userUuid, sessionToken); err != nil {
		log.Error(ctx, "Failed to register session", "error", err)
		return nilUuid, "", fmt.Errorf("failed to create session: %w", err)
	}

	return userUuid, sessionToken, nil
}

func (s *authService) Register(ctx context.Context, username, password string) (uuid.UUID, string, error) {
	var nilUuid uuid.UUID

	originalUsername := username
	normalizedUsername, errMsg := validateUsername(username, s.config.MinUsernameLength, s.config.MaxUsernameLength, s.config.UsernameAllowedSpecialChars)
	if errMsg != "" {
		log.Warn(ctx, "Registration username validation failed", "username", originalUsername)
		return nilUuid, "", &ValidationError{Message: errMsg}
	}
	username = normalizedUsername

	if errMsg := validatePassword(password, password, s.config.MinPasswordLength); errMsg != "" {
		log.Warn(ctx, "Registration password validation failed", "username", username)
		return nilUuid, "", &ValidationError{Message: errMsg}
	}

	taken, err := s.userStore.UsernameTaken(ctx, username)
	if err != nil {
		log.Error(ctx, "Failed to check if username is taken", "error", err)
		return nilUuid, "", fmt.Errorf("failed to check username availability: %w", err)
	}
	if taken {
		log.Warn(ctx, "Account creation attempt for existing user but username was taken", "username", username)
		return nilUuid, "", ErrUsernameTaken
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		log.Error(ctx, "Failed to hash password", "error", err)
		return nilUuid, "", fmt.Errorf("failed to create account: %w", err)
	}

	log.Info(ctx, "Valid registration for user", "username", username)
	userUuid, err := s.userStore.RegisterUser(ctx, username, passwordHash)
	if err != nil {
		log.Error(ctx, "Failed to register user", "error", err)
		return nilUuid, "", fmt.Errorf("failed to create account: %w", err)
	}

	sessionToken, err := generateSessionToken()
	if err != nil {
		log.Error(ctx, "Failed to generate session token", "error", err)
		return nilUuid, "", fmt.Errorf("failed to create session: %w", err)
	}

	if err := s.userStore.RegisterSession(ctx, userUuid, sessionToken); err != nil {
		log.Error(ctx, "Failed to register session", "error", err)
		return nilUuid, "", fmt.Errorf("failed to create session: %w", err)
	}

	return userUuid, sessionToken, nil
}

func (s *authService) Logout(ctx context.Context, sessionToken string) error {
	if err := s.userStore.UnRegisterSession(ctx, sessionToken); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	return nil
}

func (s *authService) ChangePassword(ctx context.Context, userUuid uuid.UUID, username, currentPassword, newPassword string) error {
	if errMsg := validatePassword(newPassword, newPassword, s.config.MinPasswordLength); errMsg != "" {
		log.Warn(ctx, "Password change validation failed", "username", username)
		return &ValidationError{Message: errMsg}
	}

	valid, err := s.validateCredentials(ctx, username, currentPassword)
	if err != nil {
		return err
	}
	if !valid {
		log.Warn(ctx, "Password change failed due to invalid current password", "username", username)
		return ErrInvalidCredentials
	}

	newPasswordHash, err := s.passwordHasher.Hash(newPassword)
	if err != nil {
		log.Error(ctx, "Failed to hash new password", "error", err)
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.userStore.UpdatePassword(ctx, username, newPasswordHash); err != nil {
		log.Error(ctx, "Failed to update password", "username", username, "error", err)
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.userStore.InvalidateAllUserSessionsExcept(ctx, userUuid, ""); err != nil {
		log.Error(ctx, "Failed to invalidate sessions after password change", "userUuid", userUuid, "error", err)
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}

	return nil
}

func (s *authService) ValidateSession(ctx context.Context, sessionToken string) (uuid.UUID, error) {
	var nilUuid uuid.UUID

	isValid, err := s.userStore.ValidateSessionToken(ctx, sessionToken)
	if err != nil {
		log.Error(ctx, "Failed to validate session token", "error", err)
		return nilUuid, fmt.Errorf("failed to validate session: %w", err)
	}
	if !isValid {
		return nilUuid, ErrInvalidCredentials
	}

	userUuid, err := s.userStore.GetUserBySessionToken(ctx, sessionToken)
	if err != nil {
		log.Error(ctx, "Failed to get user by session token", "error", err)
		return nilUuid, fmt.Errorf("failed to validate session: %w", err)
	}

	return userUuid, nil
}

func (s *authService) InvalidateOtherSessions(ctx context.Context, userUuid uuid.UUID, keepSessionToken string) error {
	if err := s.userStore.InvalidateAllUserSessionsExcept(ctx, userUuid, keepSessionToken); err != nil {
		log.Error(ctx, "Failed to invalidate sessions", "userUuid", userUuid, "error", err)
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}
	return nil
}

// validateCredentials checks a username/password pair without creating a session.
func (s *authService) validateCredentials(ctx context.Context, username, password string) (bool, error) {
	passwordHash, err := s.userStore.GetPasswordHashByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = s.passwordHasher.Compare(password, string(s.passwordHasher.DummyHash()))
			return false, nil
		}
		log.Error(ctx, "Failed to get password hash", "username", username, "error", err)
		return false, fmt.Errorf("failed to validate credentials: %w", err)
	}

	if err := s.passwordHasher.Compare(password, passwordHash); err != nil {
		return false, nil
	}
	return true, nil
}
