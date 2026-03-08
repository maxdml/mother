package coder

import (
	"testing"
)

func TestNewEngine_Defaults(t *testing.T) {
	e := New()
	if e.ReportDir != "" {
		t.Fatalf("expected empty ReportDir, got %q", e.ReportDir)
	}
}

func TestNewEngine_WithReportDir(t *testing.T) {
	e := New(WithReportDir("/tmp/reports"))
	if e.ReportDir != "/tmp/reports" {
		t.Fatalf("expected /tmp/reports, got %q", e.ReportDir)
	}
}

func TestParams_Validate_MissingProjectDir(t *testing.T) {
	p := Params{Prompt: "hello"}
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for missing project dir")
	}
}

func TestParams_Validate_MissingPrompt(t *testing.T) {
	p := Params{ProjectDir: "/tmp"}
	// No prompt is OK — interactive mode
	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParams_Validate_OK(t *testing.T) {
	p := Params{ProjectDir: "/tmp", Prompt: "hello"}
	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildClaudeCommand(t *testing.T) {
	cmd := buildClaudeCommand("/tmp/sp.txt", "/tmp/p.txt", "opus")
	if cmd == "" {
		t.Fatal("expected non-empty command")
	}
	if !contains(cmd, "--dangerously-skip-permissions") {
		t.Fatal("missing --dangerously-skip-permissions")
	}
	if !contains(cmd, "--model opus") {
		t.Fatal("missing --model flag")
	}
	if !contains(cmd, "sp.txt") {
		t.Fatal("missing system prompt file reference")
	}
	if !contains(cmd, "p.txt") {
		t.Fatal("missing prompt file reference")
	}
}

func TestBuildClaudeCommand_NoModel(t *testing.T) {
	cmd := buildClaudeCommand("/tmp/sp.txt", "/tmp/p.txt", "")
	if contains(cmd, "--model") {
		t.Fatal("should not contain --model when empty")
	}
}

func TestBuildClaudeCommand_NoPrompt(t *testing.T) {
	cmd := buildClaudeCommand("/tmp/sp.txt", "", "")
	if contains(cmd, ` -p "$(cat`) {
		t.Fatal("should not contain -p when no prompt file")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
