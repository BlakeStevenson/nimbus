package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// ConfigHandler handles configuration-related HTTP requests
type ConfigHandler struct {
	store  *configstore.Store
	logger *zap.Logger
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(store *configstore.Store, logger *zap.Logger) *ConfigHandler {
	return &ConfigHandler{
		store:  store,
		logger: logger,
	}
}

// GetConfig handles GET /api/config/{key}
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "key is required")
		return
	}

	value, err := h.store.Get(r.Context(), key)
	if err != nil {
		httputil.LogError(h.logger, err, "failed to get config", zap.String("key", key))
		httputil.RespondErrorMessage(w, http.StatusNotFound, "config not found")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"key":   key,
		"value": value,
	})
}

// SetConfig handles PUT /api/config/{key}
func (h *ConfigHandler) SetConfig(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "key is required")
		return
	}

	var body map[string]interface{}
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}

	value, ok := body["value"]
	if !ok {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "value is required")
		return
	}

	if err := h.store.Set(r.Context(), key, value); err != nil {
		httputil.LogError(h.logger, err, "failed to set config", zap.String("key", key))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to set config")
		return
	}

	// Return the stored value
	storedValue, _ := h.store.Get(r.Context(), key)
	httputil.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"key":   key,
		"value": storedValue,
	})
}

// ListConfig handles GET /api/config
func (h *ConfigHandler) ListConfig(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")

	var configs map[string]json.RawMessage
	var err error

	if prefix != "" {
		configs, err = h.store.GetByPrefix(r.Context(), prefix)
	} else {
		configs, err = h.store.GetAll(r.Context())
	}

	if err != nil {
		httputil.LogError(h.logger, err, "failed to list config")
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to list config")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, configs)
}

// DeleteConfig handles DELETE /api/config/{key}
func (h *ConfigHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "key is required")
		return
	}

	if err := h.store.Delete(r.Context(), key); err != nil {
		httputil.LogError(h.logger, err, "failed to delete config", zap.String("key", key))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to delete config")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
