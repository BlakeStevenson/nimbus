package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blakestevenson/nimbus/internal/auth/providers"
	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// service implements the Service interface
type service struct {
	queries          *generated.Queries
	jwt              *JWTManager
	passwordProvider *providers.PasswordProvider
	logger           *zap.Logger
}

// NewService creates a new authentication service
func NewService(queries *generated.Queries, jwt *JWTManager, passwordProvider *providers.PasswordProvider, logger *zap.Logger) Service {
	svc := &service{
		queries:          queries,
		jwt:              jwt,
		passwordProvider: passwordProvider,
		logger:           logger,
	}

	return svc
}

// RegisterProvider registers a new authentication provider (placeholder for future extension)
func (s *service) RegisterProvider(provider ProviderPlugin) error {
	// Future implementation for plugin system
	return nil
}

// GetProvider retrieves a registered provider by type (placeholder for future extension)
func (s *service) GetProvider(providerType string) (ProviderPlugin, error) {
	// Future implementation for plugin system
	return nil, ErrProviderNotFound
}

// Register creates a new user with the specified provider
func (s *service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	// Validate input
	if err := ValidateUsername(req.Username); err != nil {
		return nil, err
	}
	if err := ValidateEmail(req.Email); err != nil {
		return nil, err
	}

	// Validate password
	if err := providers.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Check if user already exists
	if _, err := s.queries.GetUserByUsername(ctx, req.Username); err == nil {
		return nil, ErrUserExists
	}
	if _, err := s.queries.GetUserByEmail(ctx, req.Email); err == nil {
		return nil, ErrUserExists
	}

	// Prepare metadata
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create user
	dbUser, err := s.queries.CreateUser(ctx, generated.CreateUserParams{
		Username: req.Username,
		Email:    req.Email,
		IsActive: true,
		IsAdmin:  false,
		Metadata: metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user := UserFromDB(&dbUser)

	// Create password auth provider
	if err := s.passwordProvider.CreateAuthProvider(ctx, user.ID, req.Password); err != nil {
		// Rollback user creation if provider registration fails
		_ = s.queries.DeleteUser(ctx, user.ID)
		return nil, fmt.Errorf("failed to register with provider: %w", err)
	}

	// Generate tokens
	tokens, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	s.logger.Info("user registered successfully",
		zap.Int64("user_id", user.ID),
		zap.String("username", user.Username),
		zap.String("provider", "password"),
	)

	return &AuthResponse{
		User:   user,
		Tokens: tokens,
	}, nil
}

// Login authenticates a user and returns tokens
func (s *service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	// Authenticate with password provider
	dbUser, err := s.passwordProvider.Authenticate(ctx, req.Username, req.Password)
	if err != nil {
		s.logger.Warn("authentication failed",
			zap.String("username", req.Username),
			zap.Error(err),
		)
		return nil, ErrInvalidCredentials
	}

	user := UserFromDB(dbUser)

	// Generate tokens
	tokens, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	s.logger.Info("user logged in successfully",
		zap.Int64("user_id", user.ID),
		zap.String("username", user.Username),
	)

	return &AuthResponse{
		User:   user,
		Tokens: tokens,
	}, nil
}

// RefreshToken generates a new token pair from a refresh token
func (s *service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Hash the refresh token
	tokenHash := s.jwt.HashRefreshToken(refreshToken)

	// Get refresh token from database
	dbToken, err := s.queries.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Update last used timestamp
	_ = s.queries.UpdateRefreshTokenLastUsed(ctx, dbToken.ID)

	// Get user
	user, err := s.GetUser(ctx, dbToken.UserID)
	if err != nil {
		return nil, err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Generate new tokens
	tokens, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	// Revoke old refresh token
	_ = s.queries.RevokeRefreshToken(ctx, dbToken.ID)

	s.logger.Info("tokens refreshed successfully", zap.Int64("user_id", user.ID))

	return tokens, nil
}

// ValidateToken validates an access token and returns the claims
func (s *service) ValidateToken(ctx context.Context, token string) (*Claims, error) {
	claims, err := s.jwt.ValidateAccessToken(token)
	if err != nil {
		return nil, err
	}

	// Verify user still exists and is active
	user, err := s.GetUser(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	return claims, nil
}

// RevokeToken revokes a refresh token
func (s *service) RevokeToken(ctx context.Context, refreshToken string) error {
	tokenHash := s.jwt.HashRefreshToken(refreshToken)
	return s.queries.RevokeRefreshTokenByHash(ctx, tokenHash)
}

// GetUser retrieves a user by ID
func (s *service) GetUser(ctx context.Context, userID int64) (*User, error) {
	dbUser, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return UserFromDB(&dbUser), nil
}

// GetUserByUsername retrieves a user by username
func (s *service) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	dbUser, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return UserFromDB(&dbUser), nil
}

// UpdateUser updates user information
func (s *service) UpdateUser(ctx context.Context, userID int64, updates map[string]interface{}) (*User, error) {
	params := generated.UpdateUserParams{
		ID: userID,
	}

	if username, ok := updates["username"].(string); ok {
		if err := ValidateUsername(username); err != nil {
			return nil, err
		}
		params.Username = &username
	}

	if email, ok := updates["email"].(string); ok {
		if err := ValidateEmail(email); err != nil {
			return nil, err
		}
		params.Email = &email
	}

	if isActive, ok := updates["is_active"].(bool); ok {
		params.IsActive = &isActive
	}

	if isAdmin, ok := updates["is_admin"].(bool); ok {
		params.IsAdmin = &isAdmin
	}

	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		params.Metadata = metadataJSON
	}

	dbUser, err := s.queries.UpdateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("user updated successfully", zap.Int64("user_id", userID))

	return UserFromDB(&dbUser), nil
}

// generateTokens creates access and refresh tokens for a user
func (s *service) generateTokens(ctx context.Context, user *User) (*TokenPair, error) {
	// Generate access token
	accessToken, expiresAt, err := s.jwt.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Hash and store refresh token
	tokenHash := s.jwt.HashRefreshToken(refreshToken)
	refreshExpiresAt := time.Now().Add(s.jwt.GetRefreshTokenExpiry())

	_, err = s.queries.CreateRefreshToken(ctx, generated.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: refreshExpiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}
