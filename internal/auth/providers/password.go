package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blakestevenson/nimbus/internal/db/generated"
	"golang.org/x/crypto/bcrypt"
)

const (
	ProviderTypePassword = "password"
	MinPasswordLength    = 8
	BcryptCost           = 12
)

// User interface to avoid circular imports
type User interface {
	GetID() int64
}

// userImpl is a simple implementation to pass user data
type userImpl struct {
	ID int64
}

func (u *userImpl) GetID() int64 {
	return u.ID
}

// UserFromDB converts database user to domain user (simplified to avoid import cycle)
type UserData struct {
	ID        int64
	Username  string
	Email     string
	IsActive  bool
	IsAdmin   bool
	CreatedAt interface{}
	UpdatedAt interface{}
}

// Error types (duplicated to avoid import cycle)
type AuthError string

func (e AuthError) Error() string { return string(e) }

const (
	ErrInvalidCredentials = AuthError("invalid credentials")
	ErrUserInactive       = AuthError("user account is inactive")
	ErrProviderNotFound   = AuthError("authentication provider not found")
	ErrWeakPassword       = AuthError("password does not meet requirements")
)

// PasswordProvider implements username/password authentication
type PasswordProvider struct {
	queries *generated.Queries
}

// NewPasswordProvider creates a new password authentication provider
func NewPasswordProvider(queries *generated.Queries) *PasswordProvider {
	return &PasswordProvider{
		queries: queries,
	}
}

// Type returns the provider type
func (p *PasswordProvider) Type() string {
	return ProviderTypePassword
}

// Authenticate validates username/password and returns the user data
func (p *PasswordProvider) Authenticate(ctx context.Context, username, password string) (*generated.User, error) {
	// Get user by username
	dbUser, err := p.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !dbUser.IsActive {
		return nil, ErrUserInactive
	}

	// Get password auth provider
	dbProvider, err := p.queries.GetAuthProviderByUserAndType(ctx, generated.GetAuthProviderByUserAndTypeParams{
		UserID:       dbUser.ID,
		ProviderType: ProviderTypePassword,
	})
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Extract password hash from credentials JSONB
	var creds map[string]interface{}
	if err := json.Unmarshal(dbProvider.Credentials, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	passwordHash, ok := creds["password_hash"].(string)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last used timestamp
	_ = p.queries.UpdateAuthProviderLastUsed(ctx, dbProvider.ID)

	return &dbUser, nil
}

// CreateAuthProvider creates a new password auth provider for a user
func (p *PasswordProvider) CreateAuthProvider(ctx context.Context, userID int64, password string) error {
	// Validate password strength
	if err := ValidatePassword(password); err != nil {
		return err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Store password hash in credentials JSONB
	credsMap := map[string]interface{}{
		"password_hash": string(hashedPassword),
	}
	credsJSON, err := json.Marshal(credsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Create auth provider
	_, err = p.queries.CreateAuthProvider(ctx, generated.CreateAuthProviderParams{
		UserID:       userID,
		ProviderType: ProviderTypePassword,
		ProviderID:   nil, // null for password auth
		Credentials:  credsJSON,
		Metadata:     []byte("{}"),
		IsPrimary:    true,
	})
	if err != nil {
		return fmt.Errorf("failed to create auth provider: %w", err)
	}

	return nil
}

// UpdatePassword updates the user's password
func (p *PasswordProvider) UpdatePassword(ctx context.Context, userID int64, password string) error {
	// Validate password strength
	if err := ValidatePassword(password); err != nil {
		return err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Get existing provider
	dbProvider, err := p.queries.GetAuthProviderByUserAndType(ctx, generated.GetAuthProviderByUserAndTypeParams{
		UserID:       userID,
		ProviderType: ProviderTypePassword,
	})
	if err != nil {
		return ErrProviderNotFound
	}

	// Store password hash in credentials JSONB
	credsMap := map[string]interface{}{
		"password_hash": string(hashedPassword),
	}
	credsJSON, err := json.Marshal(credsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Update credentials
	_, err = p.queries.UpdateAuthProvider(ctx, generated.UpdateAuthProviderParams{
		ID:          dbProvider.ID,
		Credentials: credsJSON,
		Metadata:    dbProvider.Metadata,
		IsPrimary:   &dbProvider.IsPrimary,
		LastUsedAt:  dbProvider.LastUsedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to update credentials: %w", err)
	}

	return nil
}

// ValidatePassword checks if a password meets requirements
func ValidatePassword(password string) error {
	if password == "" {
		return ErrInvalidCredentials
	}

	// Check minimum length
	if len(password) < MinPasswordLength {
		return ErrWeakPassword
	}

	// Check for at least one uppercase, one lowercase, and one number
	hasUpper := false
	hasLower := false
	hasNumber := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber {
		return ErrWeakPassword
	}

	return nil
}
