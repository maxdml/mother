package coder

import (
	"strings"
	"testing"
)

func TestDefaultSystemPrompt_ContainsReporting(t *testing.T) {
	p := DefaultSystemPrompt()
	if !strings.Contains(p, "report") {
		t.Fatal("default prompt should mention reporting")
	}
}

func TestDefaultSystemPrompt_ContainsSimplicity(t *testing.T) {
	p := DefaultSystemPrompt()
	if !strings.Contains(p, "simplest") {
		t.Fatal("default prompt should mention simplicity")
	}
}

func TestDefaultSystemPrompt_ContainsReproducibility(t *testing.T) {
	p := DefaultSystemPrompt()
	if !strings.Contains(p, "clean checkout") {
		t.Fatal("default prompt should mention clean checkout reproducibility")
	}
}

func TestBuildSystemPrompt_DefaultOnly(t *testing.T) {
	result := BuildSystemPrompt("")
	if result != DefaultSystemPrompt() {
		t.Fatal("with no user prompt, should return default")
	}
}

func TestBuildSystemPrompt_WithUserPrompt(t *testing.T) {
	result := BuildSystemPrompt("custom instructions")
	if !strings.Contains(result, "custom instructions") {
		t.Fatal("should contain user prompt")
	}
	if !strings.HasPrefix(result, DefaultSystemPrompt()) {
		t.Fatal("should start with default prompt")
	}
}
