package main

import (
	"strings"
	"testing"
)

func TestGenerateConfig_NoEnvVars(t *testing.T) {
	config, err := GenerateConfig("/tmp/project", "/home/user/.claude", "/home/user", nil)
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
	env := map[string]string{"API_KEY": "secret", "DEBUG": "true"}
	config, err := GenerateConfig("/tmp/project", "/home/user/.claude", "/home/user", env)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(config, "API_KEY") || !strings.Contains(config, "DEBUG") {
		t.Fatal("missing env vars in config")
	}
}

func TestGenerateConfig_DeterministicOrder(t *testing.T) {
	env := map[string]string{"Z": "1", "A": "2", "M": "3"}
	c1, _ := GenerateConfig("/tmp/p", "/tmp/c", "/tmp/h", env)
	c2, _ := GenerateConfig("/tmp/p", "/tmp/c", "/tmp/h", env)
	if c1 != c2 {
		t.Fatal("config output not deterministic")
	}
}

func TestRandomSuffix_Format(t *testing.T) {
	s := randomSuffix()
	if len(s) != 8 {
		t.Fatalf("expected 8 hex chars, got %d: %s", len(s), s)
	}
}
