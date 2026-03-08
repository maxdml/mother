package vm

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGenerateConfig_NoEnvVars(t *testing.T) {
	v := New(Config{
		ProjectDir: "/tmp/project",
		ClaudeDir:  "/home/user/.claude",
		HomeDir:    "/home/user",
	})
	config, err := v.GenerateConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(config, "/tmp/project") {
		t.Fatal("missing project dir in config")
	}
	if strings.Contains(config, "env:") {
		t.Fatal("should not contain env section with no vars")
	}
}

func TestGenerateConfig_WithEnvVars(t *testing.T) {
	v := New(Config{
		ProjectDir: "/tmp/project",
		ClaudeDir:  "/home/user/.claude",
		HomeDir:    "/home/user",
		EnvVars:    map[string]string{"API_KEY": "secret", "DEBUG": "true"},
	})
	config, err := v.GenerateConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(config, "API_KEY") || !strings.Contains(config, "DEBUG") {
		t.Fatal("missing env vars in config")
	}
}

func TestGenerateConfig_DeterministicOrder(t *testing.T) {
	cfg := Config{
		ProjectDir: "/tmp/p",
		ClaudeDir:  "/tmp/c",
		HomeDir:    "/tmp/h",
		EnvVars:    map[string]string{"Z": "1", "A": "2", "M": "3"},
	}
	v := New(cfg)
	c1, _ := v.GenerateConfig()
	c2, _ := v.GenerateConfig()
	if c1 != c2 {
		t.Fatal("config output not deterministic")
	}
}

func TestRandomName_Format(t *testing.T) {
	name := randomName()
	if !strings.HasPrefix(name, "coder-") {
		t.Fatalf("expected coder- prefix, got %s", name)
	}
	// "coder-" (6) + 8 hex chars = 14
	if len(name) != 14 {
		t.Fatalf("expected 14 chars, got %d: %s", len(name), name)
	}
}

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

	v := New(Config{
		ProjectDir: t.TempDir(),
		ClaudeDir:  t.TempDir(),
		HomeDir:    t.TempDir(),
	})
	if err := v.Start(t.Context()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer v.Cleanup()
}
