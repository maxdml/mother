package main

import (
	"encoding/json"
	"log"
	"net/http"

	"mother/control-plane/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Handler implements the generated api.ServerInterface.
type Handler struct {
	Jobs JobManager
}

var _ api.ServerInterface = (*Handler)(nil)

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req api.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	if !req.Service.Valid() {
		writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "unsupported service: " + string(req.Service)})
		return
	}

	if req.Params.ProjectDir == "" {
		writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "project_dir is required"})
		return
	}
	if req.Params.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "prompt is required"})
		return
	}

	id, err := h.Jobs.StartJob(r.Context(), string(req.Service), req.Params)
	if err != nil {
		log.Printf("failed to start job: %v", err)
		writeJSON(w, http.StatusInternalServerError, api.ErrorResponse{Error: "failed to start job"})
		return
	}

	writeJSON(w, http.StatusAccepted, api.CreateJobResponse{Id: id})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	job, err := h.Jobs.GetJob(r.Context(), id)
	if err != nil {
		log.Printf("failed to get job %s: %v", id, err)
		writeJSON(w, http.StatusInternalServerError, api.ErrorResponse{Error: "internal error"})
		return
	}
	if job == nil {
		writeJSON(w, http.StatusNotFound, api.ErrorResponse{Error: "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}
