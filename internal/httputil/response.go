package httputil

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// RespondJSON sends a JSON response
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

// RespondError sends an error response
func RespondError(w http.ResponseWriter, status int, err error, message string) {
	RespondJSON(w, status, ErrorResponse{
		Error:   err.Error(),
		Message: message,
		Code:    status,
	})
}

// RespondErrorMessage sends an error response with just a message
func RespondErrorMessage(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, ErrorResponse{
		Error: message,
		Code:  status,
	})
}

// DecodeJSON decodes a JSON request body
func DecodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// LogError logs an error with context
func LogError(logger *zap.Logger, err error, message string, fields ...zap.Field) {
	allFields := append([]zap.Field{zap.Error(err)}, fields...)
	logger.Error(message, allFields...)
}
