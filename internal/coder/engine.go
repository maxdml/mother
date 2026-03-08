package coder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxdml/mother/internal/vm"
)

// Params holds the parameters for a coder invocation.
type Params struct {
	ProjectDir   string
	Prompt       string
	PromptFile   string
	SystemPrompt string
	Model        string
	ID           string
	EnvVars      map[string]string
}

// Validate checks that required fields are set.
func (p *Params) Validate() error {
	if p.ProjectDir == "" {
		return fmt.Errorf("project_dir is required")
	}
	return nil
}

// Engine orchestrates coder invocations.
type Engine struct {
	ReportDir string
}

// Option configures an Engine.
type Option func(*Engine)

// WithReportDir sets the directory for writing reports.
func WithReportDir(dir string) Option {
	return func(e *Engine) {
		e.ReportDir = dir
	}
}

// New creates a new Engine with the given options.
func New(opts ...Option) *Engine {
	e := &Engine{}
	for _, o := range opts {
		o(e)
	}
	return e
}

// Run executes a full coder invocation: start VM, run Claude, capture output, cleanup.
// It returns a Report and any error encountered.
func (e *Engine) Run(ctx context.Context, p Params) (*Report, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	// Resolve prompt from file if provided
	prompt := p.Prompt
	if p.PromptFile != "" {
		data, err := os.ReadFile(p.PromptFile)
		if err != nil {
			return nil, fmt.Errorf("reading prompt file: %w", err)
		}
		if prompt != "" {
			prompt = prompt + "\n" + string(data)
		} else {
			prompt = string(data)
		}
	}

	// Resolve home and claude dirs
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home dir: %w", err)
	}
	claudeDir := filepath.Join(homeDir, ".claude")

	// Stage credentials
	if err := stageCredentials(homeDir, claudeDir); err != nil {
		return nil, err
	}

	// Decrypt secrets for VM injection
	secretsFile, err := vm.DecryptSecrets("coder")
	if err != nil {
		return nil, fmt.Errorf("decrypting secrets: %w", err)
	}

	// Create and start VM
	v := vm.New(vm.Config{
		ProjectDir:  p.ProjectDir,
		ClaudeDir:   claudeDir,
		HomeDir:     homeDir,
		SecretsFile: secretsFile,
		EnvVars:     p.EnvVars,
	})
	if err := v.Start(ctx); err != nil {
		return nil, fmt.Errorf("starting VM: %w", err)
	}
	defer v.Cleanup()

	// Run Claude Code inside the VM
	output, runErr := e.runClaude(ctx, v, p.ProjectDir, homeDir, prompt, p.SystemPrompt, p.Model)

	// Build report
	status := "success"
	if runErr != nil {
		status = "error"
	}

	r := Report{
		ID:      p.ID,
		Status:  status,
		Summary: output,
	}

	// Write report if we have a report dir
	if e.ReportDir != "" {
		if reportPath, wErr := WriteReport(e.ReportDir, r); wErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write report: %v\n", wErr)
		} else {
			fmt.Fprintf(os.Stderr, "Report written to %s\n", reportPath)
		}
	}

	if runErr != nil {
		return &r, runErr
	}

	return &r, nil
}

// runClaude executes Claude Code inside the VM via tmux and returns captured output.
func (e *Engine) runClaude(ctx context.Context, v *vm.VM, projectDir, homeDir, prompt, systemPrompt, model string) (string, error) {
	tmpDir := filepath.Join(homeDir, ".claude", ".coder-tmp")
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write system prompt to file
	fullSystemPrompt := BuildSystemPrompt(systemPrompt)
	spFile := filepath.Join(tmpDir, "system-prompt.txt")
	if err := os.WriteFile(spFile, []byte(fullSystemPrompt), 0600); err != nil {
		return "", fmt.Errorf("writing system prompt: %w", err)
	}

	// Write prompt to file if provided
	var promptFile string
	if prompt != "" {
		promptFile = filepath.Join(tmpDir, "prompt.txt")
		if err := os.WriteFile(promptFile, []byte(prompt), 0600); err != nil {
			return "", fmt.Errorf("writing prompt: %w", err)
		}
	}

	claudeCmd := buildClaudeCommand(spFile, promptFile, model)

	// Build tmux script
	outputLog := filepath.Join(tmpDir, "output.log")
	var script strings.Builder
	script.WriteString("#!/bin/bash\nset -e\n")
	script.WriteString(fmt.Sprintf(
		"tmux new-session -d -s coder -x 220 -y 50 "+
			"'%s 2>&1 | tee %s; tmux wait-for -S coder-done'\n",
		claudeCmd, outputLog))
	script.WriteString("tmux wait-for coder-done\n")
	script.WriteString(fmt.Sprintf("cat %s\n", outputLog))

	fmt.Fprintf(os.Stderr, "\n  To monitor the session:\n  limactl shell %s tmux attach -t coder\n\n", v.Name)

	output, err := v.RunCommand(ctx, projectDir, script.String())
	return output, err
}

// buildClaudeCommand constructs the claude CLI command string.
func buildClaudeCommand(systemPromptFile, promptFile, model string) string {
	var cmd strings.Builder
	cmd.WriteString("claude --dangerously-skip-permissions")
	cmd.WriteString(fmt.Sprintf(` --append-system-prompt "$(cat '%s')"`, systemPromptFile))
	if model != "" {
		cmd.WriteString(" --model " + model)
	}
	if promptFile != "" {
		cmd.WriteString(fmt.Sprintf(` -p "$(cat '%s')"`, promptFile))
	}
	return cmd.String()
}

// stageCredentials copies ~/.claude.json into a temp dir accessible by the VM.
func stageCredentials(homeDir, claudeDir string) error {
	coderTmpDir := filepath.Join(claudeDir, ".coder-tmp")
	if err := os.MkdirAll(coderTmpDir, 0700); err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	stagedClaudeJSON := filepath.Join(coderTmpDir, "claude.json")
	if data, err := os.ReadFile(filepath.Join(homeDir, ".claude.json")); err == nil {
		os.WriteFile(stagedClaudeJSON, data, 0600)
	}
	return nil
}
