package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/maxdml/mother/internal/coder"
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

	var rawProjectDir string
	var flagArgs []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
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

	// Build prompt
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

	// Determine report dir
	reportsDir := filepath.Join(projectDir, "reports")
	if ex, exErr := os.Executable(); exErr == nil {
		reportsDir = filepath.Join(filepath.Dir(ex), "reports")
	}

	// Create engine and run
	engine := coder.New(coder.WithReportDir(reportsDir))

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()

	_, runErr := engine.Run(ctx, coder.Params{
		ProjectDir:   projectDir,
		Prompt:       prompt,
		SystemPrompt: systemPrompt,
		Model:        model,
		ID:           invocationID,
		EnvVars:      envMap,
	})

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", runErr)
		return 1
	}

	return 0
}

func loadDotEnv(path string) map[string]string {
	env := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return env
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
		v = strings.Trim(v, `"'`)
		env[k] = v
	}
	return env
}
