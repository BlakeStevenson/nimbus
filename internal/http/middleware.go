package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/blakestevenson/nimbus/internal/auth"
	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

const (
	// ContextKeyUser is the context key for storing user claims (must be a plain string to avoid type conflicts)
	ContextKeyUser = "user"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			logger.Info("http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

// RecoverMiddleware recovers from panics and returns a 500 error
func RecoverMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						zap.Any("error", err),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
					)
					httputil.RespondErrorMessage(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "http://localhost:5173" // Default for development
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware validates JWT tokens and adds user claims to context
func AuthMiddleware(authService auth.Service, logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get token from cookie first
			var token string
			if cookie, err := r.Cookie("access_token"); err == nil {
				token = cookie.Value
			} else {
				// Fallback to Authorization header for backward compatibility
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					httputil.RespondErrorMessage(w, http.StatusUnauthorized, "missing authentication")
					return
				}

				// Check Bearer prefix
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || parts[0] != "Bearer" {
					httputil.RespondErrorMessage(w, http.StatusUnauthorized, "invalid authorization header format")
					return
				}

				token = parts[1]
			}

			// Validate token
			claims, err := authService.ValidateToken(r.Context(), token)
			if err != nil {
				logger.Warn("invalid token",
					zap.Error(err),
					zap.String("path", r.URL.Path),
				)
				httputil.RespondErrorMessage(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), ContextKeyUser, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware is like AuthMiddleware but doesn't fail if no token is provided
func OptionalAuthMiddleware(authService auth.Service, logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get token from cookie first
			var token string
			if cookie, err := r.Cookie("access_token"); err == nil {
				token = cookie.Value
			} else {
				// Fallback to Authorization header
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					next.ServeHTTP(w, r)
					return
				}

				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				}
			}

			if token != "" {
				claims, err := authService.ValidateToken(r.Context(), token)
				if err == nil {
					ctx := context.WithValue(r.Context(), ContextKeyUser, claims)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdminMiddleware checks if the user is an admin
func RequireAdminMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ContextKeyUser).(*auth.Claims)
			if !ok {
				httputil.RespondErrorMessage(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if !claims.IsAdmin {
				logger.Warn("access denied - admin required",
					zap.Int64("user_id", claims.UserID),
					zap.String("path", r.URL.Path),
				)
				httputil.RespondErrorMessage(w, http.StatusForbidden, "admin access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserClaims extracts user claims from the request context
func GetUserClaims(r *http.Request) (*auth.Claims, bool) {
	claims, ok := r.Context().Value(ContextKeyUser).(*auth.Claims)
	return claims, ok
}
