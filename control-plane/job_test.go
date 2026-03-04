package main

import (
	"sync"
	"testing"

	"mother/control-plane/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestJobStore_Create(t *testing.T) {
	store := NewJobStore()
	params := api.CoderParams{ProjectDir: "/tmp/proj", Prompt: "hello"}

	id := store.Create("coder", params)
	if id == (openapi_types.UUID{}) {
		t.Fatal("expected non-zero UUID")
	}

	job := store.Get(id)
	if job == nil {
		t.Fatal("expected job to exist")
	}
	if job.Status != api.Pending {
		t.Errorf("expected pending, got %s", job.Status)
	}
	if job.Service != "coder" {
		t.Errorf("expected coder, got %s", job.Service)
	}
	if job.Params == nil {
		t.Fatal("expected params")
	}
	if job.Params.Prompt != "hello" {
		t.Errorf("expected hello, got %s", job.Params.Prompt)
	}
	if job.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestJobStore_GetNotFound(t *testing.T) {
	store := NewJobStore()
	id := openapi_types.UUID(newUUID())
	job := store.Get(id)
	if job != nil {
		t.Error("expected nil for non-existent job")
	}
}

func TestJobStore_SetRunning(t *testing.T) {
	store := NewJobStore()
	id := store.Create("coder", api.CoderParams{ProjectDir: "/tmp", Prompt: "p"})

	if err := store.SetRunning(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := store.Get(id)
	if job.Status != api.Running {
		t.Errorf("expected running, got %s", job.Status)
	}
}

func TestJobStore_SetRunningNotFound(t *testing.T) {
	store := NewJobStore()
	id := openapi_types.UUID(newUUID())
	if err := store.SetRunning(id); err == nil {
		t.Error("expected error for non-existent job")
	}
}

func TestJobStore_SetCompleted(t *testing.T) {
	store := NewJobStore()
	id := store.Create("coder", api.CoderParams{ProjectDir: "/tmp", Prompt: "p"})
	_ = store.SetRunning(id)

	if err := store.SetCompleted(id, "output data"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := store.Get(id)
	if job.Status != api.Completed {
		t.Errorf("expected completed, got %s", job.Status)
	}
	if job.Result == nil || *job.Result != "output data" {
		t.Errorf("expected 'output data', got %v", job.Result)
	}
	if job.CompletedAt == nil || job.CompletedAt.IsZero() {
		t.Error("expected non-zero completed_at")
	}
}

func TestJobStore_SetFailed(t *testing.T) {
	store := NewJobStore()
	id := store.Create("coder", api.CoderParams{ProjectDir: "/tmp", Prompt: "p"})
	_ = store.SetRunning(id)

	if err := store.SetFailed(id, "something broke"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := store.Get(id)
	if job.Status != api.Failed {
		t.Errorf("expected failed, got %s", job.Status)
	}
	if job.Error == nil || *job.Error != "something broke" {
		t.Errorf("expected 'something broke', got %v", job.Error)
	}
	if job.CompletedAt == nil || job.CompletedAt.IsZero() {
		t.Error("expected non-zero completed_at")
	}
}

func TestJobStore_GetReturnsCopy(t *testing.T) {
	store := NewJobStore()
	id := store.Create("coder", api.CoderParams{ProjectDir: "/tmp", Prompt: "p"})

	job1 := store.Get(id)
	job1.Status = api.Completed

	job2 := store.Get(id)
	if job2.Status != api.Pending {
		t.Errorf("expected pending (store unchanged), got %s", job2.Status)
	}
}

func TestJobStore_ConcurrentAccess(t *testing.T) {
	store := NewJobStore()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := store.Create("coder", api.CoderParams{ProjectDir: "/tmp", Prompt: "p"})
			_ = store.SetRunning(id)
			_ = store.SetCompleted(id, "done")
			_ = store.Get(id)
		}()
	}
	wg.Wait()
}
