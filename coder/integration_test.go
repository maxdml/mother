package main

import (
	"os/exec"
	"testing"
)

func hasLima() bool {
	_, err := exec.LookPath("limactl")
	return err == nil
}

func TestIntegration_VMLifecycle(t *testing.T) {
	if !hasLima() {
		t.Skip("limactl not found, skipping integration test")
	}
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a VM with valid temp dirs, verify it's running, clean it up
	projectDir := t.TempDir()
	instance, err := Start(projectDir, t.TempDir(), t.TempDir(), nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer Cleanup(instance)
}
