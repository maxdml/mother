package main

import (
	"fmt"
	"sync"
	"time"

	"mother/control-plane/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// JobStore manages job state in memory. This is a stepping-stone implementation;
// DBOS workflow orchestration will replace it when the Go SDK is available.
type JobStore struct {
	mu   sync.RWMutex
	jobs map[openapi_types.UUID]*api.Job
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[openapi_types.UUID]*api.Job),
	}
}

// Create adds a new job in pending state and returns its ID.
func (s *JobStore) Create(service string, params api.CoderParams) openapi_types.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := openapi_types.UUID(newUUID())
	now := time.Now().UTC()
	job := &api.Job{
		Id:        id,
		Service:   service,
		Status:    api.Pending,
		Params:    &params,
		CreatedAt: now,
	}
	s.jobs[id] = job
	return id
}

// Get retrieves a job by ID. Returns nil if not found.
func (s *JobStore) Get(id openapi_types.UUID) *api.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil
	}
	// Return a copy to avoid races on the caller side.
	cp := *job
	return &cp
}

// SetRunning transitions a job to running state.
func (s *JobStore) SetRunning(id openapi_types.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job %s not found", id)
	}
	job.Status = api.Running
	return nil
}

// SetCompleted transitions a job to completed state with a result.
func (s *JobStore) SetCompleted(id openapi_types.UUID, result string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job %s not found", id)
	}
	job.Status = api.Completed
	job.Result = &result
	now := time.Now().UTC()
	job.CompletedAt = &now
	return nil
}

// SetFailed transitions a job to failed state with an error message.
func (s *JobStore) SetFailed(id openapi_types.UUID, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job %s not found", id)
	}
	job.Status = api.Failed
	job.Error = &errMsg
	now := time.Now().UTC()
	job.CompletedAt = &now
	return nil
}
