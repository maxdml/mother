package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// JobManager abstracts job lifecycle operations.
type JobManager interface {
	StartJob(ctx context.Context, service string, params CoderParams) (openapi_types.UUID, error)
	GetJob(ctx context.Context, id openapi_types.UUID) (*Job, error)
}

// APIHandler implements the generated ServerInterface.
type APIHandler struct {
	Jobs JobManager
}

var _ ServerInterface = (*APIHandler)(nil)

func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: Ok})
}

func (h *APIHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	if !req.Service.Valid() {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "unsupported service: " + string(req.Service)})
		return
	}

	if req.Params.ProjectDir == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "project_dir is required"})
		return
	}
	if req.Params.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "prompt is required"})
		return
	}

	id, err := h.Jobs.StartJob(r.Context(), string(req.Service), req.Params)
	if err != nil {
		log.Printf("failed to start job: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to start job"})
		return
	}

	writeJSON(w, http.StatusAccepted, CreateJobResponse{Id: id})
}

func (h *APIHandler) GetJob(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	job, err := h.Jobs.GetJob(r.Context(), id)
	if err != nil {
		log.Printf("failed to get job %s: %v", id, err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}
	if job == nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "job not found"})
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
