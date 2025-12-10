package quality

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for quality operations
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new quality handler
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Quality Definitions Handlers

func (h *Handler) ListQualityDefinitions(w http.ResponseWriter, r *http.Request) {
	definitions, err := h.service.ListQualityDefinitions(r.Context())
	if err != nil {
		h.logger.Error("failed to list quality definitions", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(definitions)
}

func (h *Handler) GetQualityDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	definition, err := h.service.GetQualityDefinition(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get quality definition", zap.Error(err))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(definition)
}

func (h *Handler) CreateQualityDefinition(w http.ResponseWriter, r *http.Request) {
	var params CreateQualityDefinitionParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	definition, err := h.service.CreateQualityDefinition(r.Context(), params)
	if err != nil {
		h.logger.Error("failed to create quality definition", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(definition)
}

func (h *Handler) UpdateQualityDefinition(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var params UpdateQualityDefinitionParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	definition, err := h.service.UpdateQualityDefinition(r.Context(), id, params)
	if err != nil {
		h.logger.Error("failed to update quality definition", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(definition)
}

func (h *Handler) DeleteQualityDefinition(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteQualityDefinition(r.Context(), id); err != nil {
		h.logger.Error("failed to delete quality definition", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Quality Profiles Handlers

func (h *Handler) ListQualityProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.service.ListQualityProfiles(r.Context())
	if err != nil {
		h.logger.Error("failed to list quality profiles", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

func (h *Handler) GetQualityProfile(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	profile, err := h.service.GetQualityProfile(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get quality profile", zap.Error(err))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *Handler) CreateQualityProfile(w http.ResponseWriter, r *http.Request) {
	var params CreateQualityProfileParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := h.service.CreateQualityProfile(r.Context(), params)
	if err != nil {
		h.logger.Error("failed to create quality profile", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

func (h *Handler) UpdateQualityProfile(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var params UpdateQualityProfileParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := h.service.UpdateQualityProfile(r.Context(), id, params)
	if err != nil {
		h.logger.Error("failed to update quality profile", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *Handler) DeleteQualityProfile(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteQualityProfile(r.Context(), id); err != nil {
		h.logger.Error("failed to delete quality profile", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Quality Detection Handler

func (h *Handler) DetectQuality(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ReleaseName string `json:"release_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	info, err := h.service.DetectQuality(r.Context(), req.ReleaseName)
	if err != nil {
		h.logger.Error("failed to detect quality", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Media Quality Handlers

func (h *Handler) GetMediaQuality(w http.ResponseWriter, r *http.Request) {

	mediaID, err := strconv.ParseInt(chi.URLParam(r, "mediaId"), 10, 64)
	if err != nil {
		http.Error(w, "invalid media id", http.StatusBadRequest)
		return
	}

	quality, err := h.service.GetMediaQuality(r.Context(), mediaID)
	if err != nil {
		h.logger.Error("failed to get media quality", zap.Error(err))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quality)
}

func (h *Handler) AssignProfileToMedia(w http.ResponseWriter, r *http.Request) {

	mediaID, err := strconv.ParseInt(chi.URLParam(r, "mediaId"), 10, 64)
	if err != nil {
		http.Error(w, "invalid media id", http.StatusBadRequest)
		return
	}

	var req struct {
		ProfileID int `json:"profile_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.AssignProfileToMedia(r.Context(), mediaID, req.ProfileID); err != nil {
		h.logger.Error("failed to assign profile", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CheckUpgrade(w http.ResponseWriter, r *http.Request) {

	mediaID, err := strconv.ParseInt(chi.URLParam(r, "mediaId"), 10, 64)
	if err != nil {
		http.Error(w, "invalid media id", http.StatusBadRequest)
		return
	}

	qualityIDStr := r.URL.Query().Get("quality_id")
	if qualityIDStr == "" {
		http.Error(w, "quality_id parameter required", http.StatusBadRequest)
		return
	}

	qualityID, err := strconv.Atoi(qualityIDStr)
	if err != nil {
		http.Error(w, "invalid quality_id", http.StatusBadRequest)
		return
	}

	result, err := h.service.CheckUpgradeAvailable(r.Context(), mediaID, qualityID)
	if err != nil {
		h.logger.Error("failed to check upgrade", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) GetUpgradeHistory(w http.ResponseWriter, r *http.Request) {

	mediaID, err := strconv.ParseInt(chi.URLParam(r, "mediaId"), 10, 64)
	if err != nil {
		http.Error(w, "invalid media id", http.StatusBadRequest)
		return
	}

	history, err := h.service.GetQualityUpgradeHistory(r.Context(), mediaID)
	if err != nil {
		h.logger.Error("failed to get upgrade history", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *Handler) ListMediaForUpgrade(w http.ResponseWriter, r *http.Request) {
	var profileID *int
	if profileIDStr := r.URL.Query().Get("profile_id"); profileIDStr != "" {
		id, err := strconv.Atoi(profileIDStr)
		if err != nil {
			http.Error(w, "invalid profile_id", http.StatusBadRequest)
			return
		}
		profileID = &id
	}

	mediaIDs, err := h.service.ListMediaForUpgrade(r.Context(), profileID)
	if err != nil {
		h.logger.Error("failed to list media for upgrade", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"media_ids": mediaIDs,
		"count":     len(mediaIDs),
	})
}
