package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"mother/control-plane/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Handler implements the generated api.ServerInterface.
type Handler struct {
	Store   *JobStore
	Service *CoderService
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

	id := h.Store.Create(string(req.Service), req.Params)

	// Run the job asynchronously.
	go h.runJob(id, req.Params)

	writeJSON(w, http.StatusAccepted, api.CreateJobResponse{Id: id})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	job := h.Store.Get(id)
	if job == nil {
		writeJSON(w, http.StatusNotFound, api.ErrorResponse{Error: "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) runJob(id openapi_types.UUID, params api.CoderParams) {
	if err := h.Store.SetRunning(id); err != nil {
		log.Printf("failed to set job %s to running: %v", id, err)
		return
	}

	result, err := h.Service.Run(context.Background(), id, params)
	if err != nil {
		if setErr := h.Store.SetFailed(id, err.Error()); setErr != nil {
			log.Printf("failed to set job %s to failed: %v", id, setErr)
		}
		return
	}

	if err := h.Store.SetCompleted(id, result); err != nil {
		log.Printf("failed to set job %s to completed: %v", id, err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}
