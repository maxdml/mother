package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv_MissingFile(t *testing.T) {
	env := loadDotEnv("/nonexistent/.env")
	if len(env) != 0 {
		t.Fatalf("expected empty map, got %v", env)
	}
}

func TestLoadDotEnv_BasicParsing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("FOO=bar\nBAZ=qux\n"), 0644)

	env := loadDotEnv(path)
	if env["FOO"] != "bar" || env["BAZ"] != "qux" {
		t.Fatalf("unexpected env: %v", env)
	}
}

func TestLoadDotEnv_CommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("# comment\n\nKEY=val\n"), 0644)

	env := loadDotEnv(path)
	if len(env) != 1 || env["KEY"] != "val" {
		t.Fatalf("unexpected env: %v", env)
	}
}

func TestLoadDotEnv_QuotedValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("A=\"hello\"\nB='world'\n"), 0644)

	env := loadDotEnv(path)
	if env["A"] != "hello" || env["B"] != "world" {
		t.Fatalf("unexpected env: %v", env)
	}
}

func TestEnvFlag_SetAndString(t *testing.T) {
	var e envFlag
	e.Set("A=1")
	e.Set("B=2")
	if e.String() != "A=1, B=2" {
		t.Fatalf("unexpected: %s", e.String())
	}
}
