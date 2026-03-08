package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func newTestUUID() openapi_types.UUID {
	uuid.SetRand(rand.Reader)
	id, err := uuid.NewRandom()
	if err != nil {
		panic(fmt.Sprintf("failed to generate UUID: %v", err))
	}
	return openapi_types.UUID(id)
}

type mockJobManager struct {
	mu     sync.Mutex
	jobs   map[string]*Job
	result string
	err    error
}

func newMockJobManager() *mockJobManager {
	return &mockJobManager{
		jobs:   make(map[string]*Job),
		result: "ok",
	}
}

func (m *mockJobManager) StartJob(ctx context.Context, service string, params CoderParams) (openapi_types.UUID, error) {
	id := newTestUUID()
	now := time.Now().UTC()
	job := &Job{
		Id:        id,
		Service:   service,
		Status:    Running,
		CreatedAt: now,
	}

	m.mu.Lock()
	m.jobs[id.String()] = job
	m.mu.Unlock()

	go func() {
		time.Sleep(10 * time.Millisecond)
		m.mu.Lock()
		defer m.mu.Unlock()
		now := time.Now().UTC()
		if m.err != nil {
			job.Status = Failed
			errStr := m.err.Error()
			job.Error = &errStr
			job.CompletedAt = &now
		} else {
			job.Status = Completed
			r := m.result
			job.Result = &r
			job.CompletedAt = &now
		}
	}()

	return id, nil
}

func (m *mockJobManager) GetJob(ctx context.Context, id openapi_types.UUID) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id.String()]
	if !ok {
		return nil, nil
	}
	cp := *job
	return &cp, nil
}

func newTestAPIHandler() (*APIHandler, *mockJobManager) {
	jobs := newMockJobManager()
	return &APIHandler{Jobs: jobs}, jobs
}

func TestHealthCheck(t *testing.T) {
	h, _ := newTestAPIHandler()
	srv := Handler(h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Status != Ok {
		t.Errorf("expected ok, got %s", resp.Status)
	}
}

func TestCreateJob_Success(t *testing.T) {
	h, _ := newTestAPIHandler()
	srv := Handler(h)

	body := `{"service":"coder","params":{"project_dir":"/tmp/proj","prompt":"do things"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp CreateJobResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Id.String() == "" {
		t.Error("expected non-empty job ID")
	}
}

func TestCreateJob_InvalidJSON(t *testing.T) {
	h, _ := newTestAPIHandler()
	srv := Handler(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateJob_MissingPrompt(t *testing.T) {
	h, _ := newTestAPIHandler()
	srv := Handler(h)

	body := `{"service":"coder","params":{"project_dir":"/tmp/proj"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateJob_MissingProjectDir(t *testing.T) {
	h, _ := newTestAPIHandler()
	srv := Handler(h)

	body := `{"service":"coder","params":{"prompt":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateJob_UnsupportedService(t *testing.T) {
	h, _ := newTestAPIHandler()
	srv := Handler(h)

	body := `{"service":"unknown","params":{"project_dir":"/tmp","prompt":"p"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetJob_NotFound(t *testing.T) {
	h, _ := newTestAPIHandler()
	srv := Handler(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/00000000-0000-0000-0000-000000000001", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestCreateAndGetJob(t *testing.T) {
	h, _ := newTestAPIHandler()
	srv := Handler(h)

	body := `{"service":"coder","params":{"project_dir":"/tmp/proj","prompt":"do things"}}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", createRec.Code)
	}

	var createResp CreateJobResponse
	if err := json.NewDecoder(createRec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+createResp.Id.String(), nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var job Job
	if err := json.NewDecoder(getRec.Body).Decode(&job); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if job.Id != createResp.Id {
		t.Errorf("expected id %s, got %s", createResp.Id, job.Id)
	}
	if job.Service != "coder" {
		t.Errorf("expected coder, got %s", job.Service)
	}
	if job.Status != Completed {
		t.Errorf("expected completed, got %s", job.Status)
	}
	if job.Result == nil || *job.Result != "ok" {
		t.Errorf("expected result 'ok', got %v", job.Result)
	}
}

func TestCreateAndGetJob_Failed(t *testing.T) {
	_, jobs := newTestAPIHandler()
	jobs.err = context.DeadlineExceeded
	h := &APIHandler{Jobs: jobs}
	srv := Handler(h)

	body := `{"service":"coder","params":{"project_dir":"/tmp/proj","prompt":"do things"}}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)

	var createResp CreateJobResponse
	_ = json.NewDecoder(createRec.Body).Decode(&createResp)

	time.Sleep(100 * time.Millisecond)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+createResp.Id.String(), nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)

	var job Job
	_ = json.NewDecoder(getRec.Body).Decode(&job)

	if job.Status != Failed {
		t.Errorf("expected failed, got %s", job.Status)
	}
	if job.Error == nil || *job.Error == "" {
		t.Error("expected non-empty error message")
	}
}
