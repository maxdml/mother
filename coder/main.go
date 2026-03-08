package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type envFlag []string

func (e *envFlag) String() string { return strings.Join(*e, ", ") }
func (e *envFlag) Set(val string) error {
	*e = append(*e, val)
	return nil
}

func main() {
	os.Exit(run())
}

// run contains the main logic and returns an exit code.
// Extracted from main() so that deferred cleanup (e.g. VM deletion) runs
// before os.Exit — os.Exit skips defers.
func run() int {
	var (
		envVars      envFlag
		printPrompt  string
		promptFile   string
		systemPrompt string
		model        string
		invocationID string
	)

	flag.Var(&envVars, "e", "Pass environment variable to the VM as KEY=VAL (repeatable)")
	flag.StringVar(&printPrompt, "p", "", "Run in print mode with the given prompt (non-interactive)")
	flag.StringVar(&printPrompt, "print", "", "Run in print mode with the given prompt (non-interactive)")
	flag.StringVar(&promptFile, "f", "", "Read the prompt from a file (used with print mode)")
	flag.StringVar(&promptFile, "prompt-file", "", "Read the prompt from a file (used with print mode)")
	flag.StringVar(&systemPrompt, "system-prompt", "", "System prompt override for Claude Code")
	flag.StringVar(&model, "model", "", "Model override (e.g., sonnet, opus)")
	flag.StringVar(&invocationID, "id", "", "Invocation ID (passed by control plane)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: coder <project-dir> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Launch Claude Code in a sandboxed Lima VM with the project directory mounted.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	// Extract the project dir (first non-flag arg) from os.Args so flags
	// can appear before or after it.
	var rawProjectDir string
	var flagArgs []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			// Consume the next arg as the flag value if this isn't a boolean flag.
			if i+1 < len(os.Args) && !strings.Contains(arg, "=") {
				i++
				flagArgs = append(flagArgs, os.Args[i])
			}
		} else if rawProjectDir == "" {
			rawProjectDir = arg
		}
	}
	flag.CommandLine.Parse(flagArgs)

	if rawProjectDir == "" {
		flag.Usage()
		return 1
	}

	projectDir, err := filepath.Abs(rawProjectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving project dir: %v\n", err)
		return 1
	}

	info, err := os.Stat(projectDir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a valid directory\n", projectDir)
		return 1
	}

	// Load .env from project dir
	envMap := loadDotEnv(filepath.Join(projectDir, ".env"))

	// Merge CLI -e flags (CLI wins on conflict)
	for _, e := range envVars {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: ignoring malformed env var %q (expected KEY=VAL)\n", e)
			continue
		}
		envMap[k] = v
	}

	// Build the prompt for print mode
	var prompt string
	if printPrompt != "" || promptFile != "" {
		var parts []string
		if printPrompt != "" {
			parts = append(parts, printPrompt)
		}
		if promptFile != "" {
			data, err := os.ReadFile(promptFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading prompt file: %v\n", err)
				return 1
			}
			parts = append(parts, string(data))
		}
		prompt = strings.Join(parts, "\n")
	}

	// Resolve ~/.claude for OAuth credentials (Claude Max)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving home dir: %v\n", err)
		return 1
	}
	claudeDir := filepath.Join(homeDir, ".claude")

	// Stage .claude.json into the claude dir so the VM provision script can
	// access it without mounting all of $HOME (which would conflict with the
	// writable project-dir mount when the project is under $HOME).
	coderTmpDir := filepath.Join(claudeDir, ".coder-tmp")
	if err := os.MkdirAll(coderTmpDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		return 1
	}
	stagedClaudeJSON := filepath.Join(coderTmpDir, "claude.json")
	if data, err := os.ReadFile(filepath.Join(homeDir, ".claude.json")); err == nil {
		os.WriteFile(stagedClaudeJSON, data, 0600)
	}

	// Run the VM
	instance, err := Start(projectDir, claudeDir, homeDir, envMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting VM: %v\n", err)
		return 1
	}

	setupCleanup(instance)
	defer Cleanup(instance)

	output, runErr := RunClaude(instance, projectDir, homeDir, prompt, systemPrompt, model)

	// Write report
	reportsDir := filepath.Join(projectDir, "reports")
	if ex, exErr := os.Executable(); exErr == nil {
		reportsDir = filepath.Join(filepath.Dir(ex), "reports")
	}

	status := "success"
	if runErr != nil {
		status = "error"
	}

	r := Report{
		ID:      invocationID,
		Status:  status,
		Summary: output,
	}
	if reportPath, wErr := WriteReport(reportsDir, r); wErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write report: %v\n", wErr)
	} else {
		fmt.Fprintf(os.Stderr, "Report written to %s\n", reportPath)
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", runErr)
		return 1
	}

	return 0
}

// loadDotEnv parses a simple KEY=VAL .env file. Blank lines and # comments are skipped.
func loadDotEnv(path string) map[string]string {
	env := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return env // missing .env is fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		// Strip optional surrounding quotes from value
		v = strings.Trim(v, `"'`)
		env[k] = v
	}
	return env
}
