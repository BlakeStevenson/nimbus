package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/blakestevenson/nimbus/internal/db/generated"
)

// User represents an authenticated user
type User struct {
	ID        int64                  `json:"id"`
	Username  string                 `json:"username"`
	Email     string                 `json:"email"`
	IsActive  bool                   `json:"is_active"`
	IsAdmin   bool                   `json:"is_admin"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// AuthProvider represents an authentication provider for a user
type AuthProvider struct {
	ID           int64                  `json:"id"`
	UserID       int64                  `json:"user_id"`
	ProviderType string                 `json:"provider_type"`
	ProviderID   *string                `json:"provider_id,omitempty"`
	IsPrimary    bool                   `json:"is_primary"`
	LastUsedAt   *time.Time             `json:"last_used_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// Claims represents JWT token claims
type Claims struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	IsAdmin   bool   `json:"is_admin"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

// RegisterRequest contains user registration data
type RegisterRequest struct {
	Username     string                 `json:"username"`
	Email        string                 `json:"email"`
	Password     string                 `json:"password"`
	ProviderType string                 `json:"provider_type,omitempty"` // defaults to "password"
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// LoginRequest contains login credentials
type LoginRequest struct {
	Username     string                 `json:"username"`
	Password     string                 `json:"password"`
	ProviderType string                 `json:"provider_type,omitempty"` // defaults to "password"
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RefreshRequest contains refresh token
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthResponse contains authentication response data
type AuthResponse struct {
	User   *User      `json:"user"`
	Tokens *TokenPair `json:"tokens"`
}

// ProviderPlugin defines the interface for authentication providers
type ProviderPlugin interface {
	// Type returns the provider type identifier (e.g., "password", "oauth", "saml")
	Type() string

	// Authenticate validates credentials and returns the user if successful
	Authenticate(ctx context.Context, credentials map[string]interface{}) (*User, error)

	// Register creates a new user with this provider
	Register(ctx context.Context, user *User, credentials map[string]interface{}) error

	// UpdateCredentials updates the user's credentials for this provider
	UpdateCredentials(ctx context.Context, userID int64, credentials map[string]interface{}) error

	// ValidateCredentials checks if the credentials are valid without authenticating
	ValidateCredentials(credentials map[string]interface{}) error
}

// Service defines the authentication service interface
type Service interface {
	// Register creates a new user with the specified provider
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)

	// Login authenticates a user and returns tokens
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)

	// RefreshToken generates a new token pair from a refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)

	// ValidateToken validates an access token and returns the claims
	ValidateToken(ctx context.Context, token string) (*Claims, error)

	// RevokeToken revokes a refresh token
	RevokeToken(ctx context.Context, refreshToken string) error

	// GetUser retrieves a user by ID
	GetUser(ctx context.Context, userID int64) (*User, error)

	// GetUserByUsername retrieves a user by username
	GetUserByUsername(ctx context.Context, username string) (*User, error)

	// UpdateUser updates user information
	UpdateUser(ctx context.Context, userID int64, updates map[string]interface{}) (*User, error)

	// RegisterProvider registers a new authentication provider plugin
	RegisterProvider(provider ProviderPlugin) error

	// GetProvider retrieves a registered provider by type
	GetProvider(providerType string) (ProviderPlugin, error)
}

// Helper function to convert DB user to domain user
func UserFromDB(dbUser *generated.User) *User {
	metadata := make(map[string]interface{})
	if len(dbUser.Metadata) > 0 {
		_ = json.Unmarshal(dbUser.Metadata, &metadata)
	}

	return &User{
		ID:        dbUser.ID,
		Username:  dbUser.Username,
		Email:     dbUser.Email,
		IsActive:  dbUser.IsActive,
		IsAdmin:   dbUser.IsAdmin,
		Metadata:  metadata,
		CreatedAt: dbUser.CreatedAt.Time,
		UpdatedAt: dbUser.UpdatedAt.Time,
	}
}

// Helper function to convert DB auth provider to domain auth provider
func AuthProviderFromDB(dbProvider *generated.AuthProvider) *AuthProvider {
	metadata := make(map[string]interface{})
	if len(dbProvider.Metadata) > 0 {
		_ = json.Unmarshal(dbProvider.Metadata, &metadata)
	}

	var lastUsedAt *time.Time
	if dbProvider.LastUsedAt.Valid {
		lastUsedAt = &dbProvider.LastUsedAt.Time
	}

	return &AuthProvider{
		ID:           dbProvider.ID,
		UserID:       dbProvider.UserID,
		ProviderType: dbProvider.ProviderType,
		ProviderID:   dbProvider.ProviderID,
		IsPrimary:    dbProvider.IsPrimary,
		LastUsedAt:   lastUsedAt,
		Metadata:     metadata,
		CreatedAt:    dbProvider.CreatedAt.Time,
		UpdatedAt:    dbProvider.UpdatedAt.Time,
	}
}
