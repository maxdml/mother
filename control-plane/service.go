package main

import (
	"bytes"
	"context"
	"os/exec"

	"mother/control-plane/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// CommandExecutor abstracts command execution for testing.
type CommandExecutor interface {
	Execute(ctx context.Context, name string, args []string, env []string) (stdout string, stderr string, err error)
}

// RealExecutor runs commands via os/exec.
type RealExecutor struct{}

func (e *RealExecutor) Execute(ctx context.Context, name string, args []string, env []string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// CoderService manages execution of the coder binary.
type CoderService struct {
	BinaryPath string
	Executor   CommandExecutor
}

func NewCoderService(binaryPath string) *CoderService {
	return &CoderService{
		BinaryPath: binaryPath,
		Executor:   &RealExecutor{},
	}
}

// BuildArgs constructs CLI arguments for the coder binary from job parameters.
func (s *CoderService) BuildArgs(jobID openapi_types.UUID, params api.CoderParams) []string {
	args := []string{
		"--id", jobID.String(),
		"-p", params.Prompt,
		params.ProjectDir,
	}

	if params.SystemPrompt != nil && *params.SystemPrompt != "" {
		args = append([]string{"--system-prompt", *params.SystemPrompt}, args...)
	}
	if params.Model != nil && *params.Model != "" {
		args = append([]string{"--model", *params.Model}, args...)
	}

	return args
}

// BuildEnv constructs the environment variables for the coder process.
func (s *CoderService) BuildEnv(params api.CoderParams) []string {
	if params.EnvVars == nil {
		return nil
	}
	env := make([]string, 0, len(*params.EnvVars))
	for k, v := range *params.EnvVars {
		env = append(env, k+"="+v)
	}
	return env
}

// Run executes the coder binary with the given job parameters and returns stdout.
func (s *CoderService) Run(ctx context.Context, jobID openapi_types.UUID, params api.CoderParams) (string, error) {
	args := s.BuildArgs(jobID, params)
	env := s.BuildEnv(params)
	stdout, stderr, err := s.Executor.Execute(ctx, s.BinaryPath, args, env)
	if err != nil {
		if stderr != "" {
			return "", &CoderError{Err: err, Stderr: stderr}
		}
		return "", err
	}
	return stdout, nil
}

// CoderError wraps an execution error with stderr output.
type CoderError struct {
	Err    error
	Stderr string
}

func (e *CoderError) Error() string {
	return e.Err.Error() + ": " + e.Stderr
}

func (e *CoderError) Unwrap() error {
	return e.Err
}
