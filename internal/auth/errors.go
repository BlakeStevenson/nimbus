package auth

import "errors"

var (
	// ErrInvalidCredentials is returned when authentication fails
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserNotFound is returned when a user cannot be found
	ErrUserNotFound = errors.New("user not found")

	// ErrUserExists is returned when trying to create a user that already exists
	ErrUserExists = errors.New("user already exists")

	// ErrInvalidToken is returned when a token is invalid or expired
	ErrInvalidToken = errors.New("invalid or expired token")

	// ErrTokenRevoked is returned when a token has been revoked
	ErrTokenRevoked = errors.New("token has been revoked")

	// ErrProviderNotFound is returned when a provider cannot be found
	ErrProviderNotFound = errors.New("authentication provider not found")

	// ErrProviderExists is returned when trying to register a provider that already exists
	ErrProviderExists = errors.New("authentication provider already exists")

	// ErrInvalidProviderType is returned when the provider type is invalid
	ErrInvalidProviderType = errors.New("invalid provider type")

	// ErrUserInactive is returned when trying to authenticate an inactive user
	ErrUserInactive = errors.New("user account is inactive")

	// ErrWeakPassword is returned when a password doesn't meet requirements
	ErrWeakPassword = errors.New("password does not meet requirements")

	// ErrInvalidEmail is returned when an email is invalid
	ErrInvalidEmail = errors.New("invalid email address")

	// ErrInvalidUsername is returned when a username is invalid
	ErrInvalidUsername = errors.New("invalid username")
)
