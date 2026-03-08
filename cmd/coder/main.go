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
	"time"

	"github.com/maxdml/mother/api"
	"github.com/maxdml/mother/internal/coder"
	"github.com/maxdml/mother/internal/workflow"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
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

	// Initialize DBOS
	databaseURL := os.Getenv("DBOS_DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintf(os.Stderr, "Error: DBOS_DATABASE_URL environment variable is required\n")
		return 1
	}

	workflow.CoderEngine = coder.New()

	dbosCtx, err := dbos.NewDBOSContext(context.Background(), dbos.Config{
		DatabaseURL:     databaseURL,
		AppName:         "mother",
		ConductorAPIKey: os.Getenv("DBOS_CONDUCTOR_API_KEY"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing DBOS: %v\n", err)
		return 1
	}

	dbos.RegisterWorkflow(dbosCtx, workflow.CoderWorkflow)

	if err := dbos.Launch(dbosCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Error launching DBOS: %v\n", err)
		return 1
	}
	defer dbos.Shutdown(dbosCtx, 5*time.Second)

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()

	// Start job via DBOS workflow
	jobs := workflow.NewDBOSJobManager(dbosCtx)

	coderParams := api.CoderParams{
		ProjectDir:   projectDir,
		Prompt:       prompt,
		SystemPrompt: strPtr(systemPrompt),
		Model:        strPtr(model),
		EnvVars:      mapPtr(envMap),
	}

	jobID, err := jobs.StartJob(ctx, "coder", coderParams)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting job: %v\n", err)
		return 1
	}

	fmt.Printf("Job started: %s\n", jobID)

	// Poll until completion
	var lastStatus api.JobStatus
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "Interrupted\n")
			return 1
		case <-ticker.C:
			job, err := jobs.GetJob(ctx, jobID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error polling job: %v\n", err)
				return 1
			}
			if job == nil {
				fmt.Fprintf(os.Stderr, "Error: job not found\n")
				return 1
			}

			if job.Status != lastStatus {
				fmt.Printf("Status: %s\n", job.Status)
				lastStatus = job.Status
			}

			switch job.Status {
			case api.Completed:
				if job.Result != nil {
					fmt.Println(*job.Result)
				}
				return 0
			case api.Failed:
				if job.Error != nil {
					fmt.Fprintf(os.Stderr, "Error: %s\n", *job.Error)
				} else {
					fmt.Fprintf(os.Stderr, "Job failed\n")
				}
				return 1
			}
		}
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func mapPtr(m map[string]string) *map[string]string {
	if len(m) == 0 {
		return nil
	}
	return &m
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
