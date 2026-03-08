package workflow

import (
	"testing"

	"github.com/maxdml/mother/api"
)

func TestDerefStr_Nil(t *testing.T) {
	if got := derefStr(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestDerefStr_Value(t *testing.T) {
	s := "hello"
	if got := derefStr(&s); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}

func TestDerefMap_Nil(t *testing.T) {
	if got := derefMap(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestDerefMap_Value(t *testing.T) {
	m := map[string]string{"a": "b"}
	got := derefMap(&m)
	if got["a"] != "b" {
		t.Fatalf("expected b, got %v", got)
	}
}

func TestNewUUID_Unique(t *testing.T) {
	a := newUUID()
	b := newUUID()
	if a == b {
		t.Fatal("expected unique UUIDs")
	}
}

func TestDBOSJobManager_ImplementsInterface(t *testing.T) {
	// Compile-time check already exists via var _ api.JobManager = (*DBOSJobManager)(nil)
	// This test just verifies the import works
	var _ api.JobManager = (*DBOSJobManager)(nil)
}
