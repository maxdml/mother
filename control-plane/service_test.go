package main

import (
	"context"
	"errors"
	"testing"

	"mother/control-plane/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// MockExecutor records the command and returns preset values.
type MockExecutor struct {
	CalledName string
	CalledArgs []string
	CalledEnv  []string
	Stdout     string
	Stderr     string
	Err        error
}

func (m *MockExecutor) Execute(_ context.Context, name string, args []string, env []string) (string, string, error) {
	m.CalledName = name
	m.CalledArgs = args
	m.CalledEnv = env
	return m.Stdout, m.Stderr, m.Err
}

func TestCoderService_BuildArgs_Basic(t *testing.T) {
	svc := NewCoderService("/usr/local/bin/coder")
	id := openapi_types.UUID(newUUID())
	params := api.CoderParams{
		ProjectDir: "/home/user/project",
		Prompt:     "fix the tests",
	}

	args := svc.BuildArgs(id, params)
	expected := []string{"--id", id.String(), "-p", "fix the tests", "/home/user/project"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, v := range expected {
		if args[i] != v {
			t.Errorf("arg[%d]: expected %q, got %q", i, v, args[i])
		}
	}
}

func TestCoderService_BuildArgs_WithOptional(t *testing.T) {
	svc := NewCoderService("/usr/local/bin/coder")
	id := openapi_types.UUID(newUUID())
	model := "opus"
	sysprompt := "you are helpful"
	params := api.CoderParams{
		ProjectDir:   "/home/user/project",
		Prompt:       "do stuff",
		Model:        &model,
		SystemPrompt: &sysprompt,
	}

	args := svc.BuildArgs(id, params)

	// Check model and system-prompt flags are present
	found := map[string]string{}
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--model" {
			found["model"] = args[i+1]
		}
		if args[i] == "--system-prompt" {
			found["system-prompt"] = args[i+1]
		}
	}
	if found["model"] != "opus" {
		t.Errorf("expected model=opus, got %q", found["model"])
	}
	if found["system-prompt"] != "you are helpful" {
		t.Errorf("expected system-prompt='you are helpful', got %q", found["system-prompt"])
	}
}

func TestCoderService_BuildEnv(t *testing.T) {
	svc := NewCoderService("/usr/local/bin/coder")

	t.Run("nil env_vars", func(t *testing.T) {
		params := api.CoderParams{ProjectDir: "/tmp", Prompt: "p"}
		env := svc.BuildEnv(params)
		if env != nil {
			t.Errorf("expected nil, got %v", env)
		}
	})

	t.Run("with env_vars", func(t *testing.T) {
		vars := map[string]string{"FOO": "bar", "BAZ": "qux"}
		params := api.CoderParams{ProjectDir: "/tmp", Prompt: "p", EnvVars: &vars}
		env := svc.BuildEnv(params)
		if len(env) != 2 {
			t.Fatalf("expected 2 env vars, got %d", len(env))
		}
		envMap := map[string]bool{}
		for _, e := range env {
			envMap[e] = true
		}
		if !envMap["FOO=bar"] || !envMap["BAZ=qux"] {
			t.Errorf("unexpected env: %v", env)
		}
	})
}

func TestCoderService_Run_Success(t *testing.T) {
	mock := &MockExecutor{Stdout: "job output"}
	svc := &CoderService{BinaryPath: "/bin/coder", Executor: mock}
	id := openapi_types.UUID(newUUID())
	params := api.CoderParams{ProjectDir: "/tmp", Prompt: "hello"}

	result, err := svc.Run(context.Background(), id, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "job output" {
		t.Errorf("expected 'job output', got %q", result)
	}
	if mock.CalledName != "/bin/coder" {
		t.Errorf("expected /bin/coder, got %s", mock.CalledName)
	}
}

func TestCoderService_Run_Error(t *testing.T) {
	mock := &MockExecutor{Err: errors.New("exit 1"), Stderr: "boom"}
	svc := &CoderService{BinaryPath: "/bin/coder", Executor: mock}
	id := openapi_types.UUID(newUUID())
	params := api.CoderParams{ProjectDir: "/tmp", Prompt: "hello"}

	_, err := svc.Run(context.Background(), id, params)
	if err == nil {
		t.Fatal("expected error")
	}

	var ce *CoderError
	if !errors.As(err, &ce) {
		t.Fatalf("expected CoderError, got %T", err)
	}
	if ce.Stderr != "boom" {
		t.Errorf("expected stderr 'boom', got %q", ce.Stderr)
	}
}

func TestCoderService_Run_ErrorNoStderr(t *testing.T) {
	mock := &MockExecutor{Err: errors.New("exit 1")}
	svc := &CoderService{BinaryPath: "/bin/coder", Executor: mock}
	id := openapi_types.UUID(newUUID())
	params := api.CoderParams{ProjectDir: "/tmp", Prompt: "hello"}

	_, err := svc.Run(context.Background(), id, params)
	if err == nil {
		t.Fatal("expected error")
	}
	// Should be a plain error, not CoderError
	var ce *CoderError
	if errors.As(err, &ce) {
		t.Error("expected plain error when no stderr")
	}
}
