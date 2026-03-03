package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Report struct {
	ID        string   `json:"id"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Output    string   `json:"output,omitempty"`
	Tradeoffs []string `json:"tradeoffs,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// WriteReport writes a JSON report to the given directory.
// If r.ID is empty, a timestamp-based name is used.
func WriteReport(dir string, r Report) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating reports dir: %w", err)
	}

	if r.Timestamp == "" {
		r.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	name := r.ID
	if name == "" {
		name = fmt.Sprintf("coder-%d", time.Now().UnixMilli())
	}

	path := filepath.Join(dir, name+".json")
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("writing report: %w", err)
	}

	return path, nil
}
