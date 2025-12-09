package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// DefaultAccessTokenExpiry is the default expiry time for access tokens
	DefaultAccessTokenExpiry = 15 * time.Minute

	// DefaultRefreshTokenExpiry is the default expiry time for refresh tokens
	DefaultRefreshTokenExpiry = 7 * 24 * time.Hour
)

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secretKey          []byte
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
	if accessExpiry == 0 {
		accessExpiry = DefaultAccessTokenExpiry
	}
	if refreshExpiry == 0 {
		refreshExpiry = DefaultRefreshTokenExpiry
	}

	return &JWTManager{
		secretKey:          []byte(secretKey),
		accessTokenExpiry:  accessExpiry,
		refreshTokenExpiry: refreshExpiry,
	}
}

// GenerateAccessToken creates a new JWT access token
func (jm *JWTManager) GenerateAccessToken(user *User) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(jm.accessTokenExpiry)

	claims := Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		IsAdmin:   user.IsAdmin,
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt.Unix(),
	}

	token, err := jm.generateToken(claims)
	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

// GenerateRefreshToken creates a new refresh token (random string)
func (jm *JWTManager) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateAccessToken validates a JWT access token and returns the claims
func (jm *JWTManager) ValidateAccessToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Decode header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrInvalidToken
	}

	// Check algorithm
	if alg, ok := header["alg"].(string); !ok || alg != "HS256" {
		return nil, ErrInvalidToken
	}

	// Verify signature
	signature := parts[0] + "." + parts[1]
	expectedSignature := jm.sign(signature)
	if parts[2] != expectedSignature {
		return nil, ErrInvalidToken
	}

	// Decode payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrInvalidToken
	}

	return &claims, nil
}

// HashRefreshToken hashes a refresh token for storage
func (jm *JWTManager) HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// GetRefreshTokenExpiry returns the refresh token expiry duration
func (jm *JWTManager) GetRefreshTokenExpiry() time.Duration {
	return jm.refreshTokenExpiry
}

// generateToken creates a JWT token with the given claims
func (jm *JWTManager) generateToken(claims Claims) (string, error) {
	// Create header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Create payload
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Create signature
	signature := headerEncoded + "." + payloadEncoded
	signatureEncoded := jm.sign(signature)

	return signature + "." + signatureEncoded, nil
}

// sign creates an HMAC signature for the given data
func (jm *JWTManager) sign(data string) string {
	h := hmac.New(sha256.New, jm.secretKey)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
