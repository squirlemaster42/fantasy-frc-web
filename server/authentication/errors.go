package authentication

import "errors"

var (
	// ErrInvalidCredentials is returned when a username/password pair does not match.
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrUsernameTaken is returned when attempting to register a username that already exists.
	ErrUsernameTaken = errors.New("username is already taken")
)

// ValidationError carries a user-facing message for a validation failure.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// IsValidationError reports whether err is a ValidationError.
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}
