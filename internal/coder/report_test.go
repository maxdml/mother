package coder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReport_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	r := Report{
		ID:      "test-123",
		Status:  "success",
		Summary: "Built the thing",
	}
	path, err := WriteReport(dir, r)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "test-123" || got.Status != "success" {
		t.Fatalf("unexpected report: %+v", got)
	}
}

func TestWriteReport_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "reports")
	r := Report{ID: "abc", Status: "success", Summary: "ok"}
	_, err := WriteReport(dir, r)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWriteReport_DefaultsIDWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	r := Report{Status: "success", Summary: "no id"}
	path, err := WriteReport(dir, r)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) == ".json" {
		t.Fatal("filename should not be empty")
	}
}
