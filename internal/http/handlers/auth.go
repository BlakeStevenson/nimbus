package handlers

import (
	"errors"
	"net/http"

	"github.com/blakestevenson/nimbus/internal/auth"
	"github.com/blakestevenson/nimbus/internal/httputil"
	"go.uber.org/zap"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService auth.Service
	logger      *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService auth.Service, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.authService.Register(r.Context(), req)
	if err != nil {
		h.handleAuthError(w, err, "registration failed")
		return
	}

	// Set httpOnly cookies for tokens
	h.setTokenCookies(w, response.Tokens)

	// Return response without tokens (they're in cookies now)
	httputil.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"user": response.User,
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.authService.Login(r.Context(), req)
	if err != nil {
		h.handleAuthError(w, err, "login failed")
		return
	}

	// Set httpOnly cookies for tokens
	h.setTokenCookies(w, response.Tokens)

	// Return response without tokens (they're in cookies now)
	httputil.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"user": response.User,
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	tokens, err := h.authService.RefreshToken(r.Context(), refreshCookie.Value)
	if err != nil {
		h.handleAuthError(w, err, "token refresh failed")
		return
	}

	// Set new tokens in httpOnly cookies
	h.setTokenCookies(w, tokens)

	httputil.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "tokens refreshed successfully",
	})
}

// Logout handles user logout (revokes refresh token)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshCookie, err := r.Cookie("refresh_token")
	if err == nil {
		// Revoke the refresh token if it exists
		if err := h.authService.RevokeToken(r.Context(), refreshCookie.Value); err != nil {
			h.logger.Warn("failed to revoke token", zap.Error(err))
			// Don't fail the logout request even if token revocation fails
		}
	}

	// Clear cookies
	h.clearTokenCookies(w)

	httputil.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// Me returns the current user's information
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := getUserClaims(r)
	if !ok {
		httputil.RespondErrorMessage(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.authService.GetUser(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("failed to get user", zap.Error(err), zap.Int64("user_id", claims.UserID))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to get user information")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, user)
}

// UpdateProfile updates the current user's profile
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := getUserClaims(r)
	if !ok {
		httputil.RespondErrorMessage(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var updates map[string]interface{}
	if err := httputil.DecodeJSON(r, &updates); err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Prevent users from modifying admin status themselves
	delete(updates, "is_admin")
	delete(updates, "is_active")

	user, err := h.authService.UpdateUser(r.Context(), claims.UserID, updates)
	if err != nil {
		h.handleAuthError(w, err, "failed to update profile")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, user)
}

// handleAuthError maps authentication errors to HTTP responses
func (h *AuthHandler) handleAuthError(w http.ResponseWriter, err error, defaultMsg string) {
	h.logger.Warn(defaultMsg, zap.Error(err))

	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		httputil.RespondErrorMessage(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, auth.ErrUserExists):
		httputil.RespondErrorMessage(w, http.StatusConflict, "user already exists")
	case errors.Is(err, auth.ErrUserNotFound):
		httputil.RespondErrorMessage(w, http.StatusNotFound, "user not found")
	case errors.Is(err, auth.ErrInvalidToken):
		httputil.RespondErrorMessage(w, http.StatusUnauthorized, "invalid or expired token")
	case errors.Is(err, auth.ErrTokenRevoked):
		httputil.RespondErrorMessage(w, http.StatusUnauthorized, "token has been revoked")
	case errors.Is(err, auth.ErrUserInactive):
		httputil.RespondErrorMessage(w, http.StatusForbidden, "user account is inactive")
	case errors.Is(err, auth.ErrWeakPassword):
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "password does not meet requirements: minimum 8 characters with uppercase, lowercase, and number")
	case errors.Is(err, auth.ErrInvalidEmail):
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "invalid email address")
	case errors.Is(err, auth.ErrInvalidUsername):
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "invalid username: must be 3-32 characters, alphanumeric with underscores and hyphens only")
	case errors.Is(err, auth.ErrProviderNotFound):
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "authentication provider not found")
	default:
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, defaultMsg)
	}
}

// setTokenCookies sets httpOnly cookies for access and refresh tokens
func (h *AuthHandler) setTokenCookies(w http.ResponseWriter, tokens *auth.TokenPair) {
	// Set access token cookie (15 minutes)
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokens.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Only send over HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   15 * 60, // 15 minutes in seconds
	})

	// Set refresh token cookie (7 days)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Only send over HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days in seconds
	})
}

// clearTokenCookies clears the authentication cookies
func (h *AuthHandler) clearTokenCookies(w http.ResponseWriter) {
	// Clear access token
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // Expire immediately
	})

	// Clear refresh token
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // Expire immediately
	})
}
