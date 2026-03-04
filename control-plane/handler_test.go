package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mother/control-plane/api"
)

func newTestHandler() (*Handler, *MockExecutor) {
	mock := &MockExecutor{Stdout: "ok"}
	store := NewJobStore()
	svc := &CoderService{BinaryPath: "/bin/coder", Executor: mock}
	h := &Handler{Store: store, Service: svc}
	return h, mock
}

func TestHealthCheck(t *testing.T) {
	h, _ := newTestHandler()
	srv := api.Handler(h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp api.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Status != api.Ok {
		t.Errorf("expected ok, got %s", resp.Status)
	}
}

func TestCreateJob_Success(t *testing.T) {
	h, _ := newTestHandler()
	srv := api.Handler(h)

	body := `{"service":"coder","params":{"project_dir":"/tmp/proj","prompt":"do things"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp api.CreateJobResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Id.String() == "" {
		t.Error("expected non-empty job ID")
	}
}

func TestCreateJob_InvalidJSON(t *testing.T) {
	h, _ := newTestHandler()
	srv := api.Handler(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateJob_MissingPrompt(t *testing.T) {
	h, _ := newTestHandler()
	srv := api.Handler(h)

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
	h, _ := newTestHandler()
	srv := api.Handler(h)

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
	h, _ := newTestHandler()
	srv := api.Handler(h)

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
	h, _ := newTestHandler()
	srv := api.Handler(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/00000000-0000-0000-0000-000000000001", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestCreateAndGetJob(t *testing.T) {
	h, _ := newTestHandler()
	srv := api.Handler(h)

	// Create a job
	body := `{"service":"coder","params":{"project_dir":"/tmp/proj","prompt":"do things"}}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", createRec.Code)
	}

	var createResp api.CreateJobResponse
	if err := json.NewDecoder(createRec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Give the async goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	// Get the job
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+createResp.Id.String(), nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var job api.Job
	if err := json.NewDecoder(getRec.Body).Decode(&job); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if job.Id != createResp.Id {
		t.Errorf("expected id %s, got %s", createResp.Id, job.Id)
	}
	if job.Service != "coder" {
		t.Errorf("expected coder, got %s", job.Service)
	}
	// The mock returns "ok" immediately, so it should be completed
	if job.Status != api.Completed {
		t.Errorf("expected completed, got %s", job.Status)
	}
	if job.Result == nil || *job.Result != "ok" {
		t.Errorf("expected result 'ok', got %v", job.Result)
	}
}

func TestCreateAndGetJob_Failed(t *testing.T) {
	h, mock := newTestHandler()
	mock.Stdout = ""
	mock.Stderr = "process died"
	mock.Err = context.DeadlineExceeded
	srv := api.Handler(h)

	body := `{"service":"coder","params":{"project_dir":"/tmp/proj","prompt":"do things"}}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)

	var createResp api.CreateJobResponse
	_ = json.NewDecoder(createRec.Body).Decode(&createResp)

	time.Sleep(100 * time.Millisecond)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+createResp.Id.String(), nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)

	var job api.Job
	_ = json.NewDecoder(getRec.Body).Decode(&job)

	if job.Status != api.Failed {
		t.Errorf("expected failed, got %s", job.Status)
	}
	if job.Error == nil || *job.Error == "" {
		t.Error("expected non-empty error message")
	}
}
