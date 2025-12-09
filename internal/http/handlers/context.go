package handlers

import (
	"net/http"

	"github.com/blakestevenson/nimbus/internal/auth"
)

// getUserClaims extracts user claims from the request context
// Note: Must use the same context key string as middleware.go ("user")
func getUserClaims(r *http.Request) (*auth.Claims, bool) {
	claims, ok := r.Context().Value("user").(*auth.Claims)
	return claims, ok
}
