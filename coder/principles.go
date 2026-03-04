package main

const defaultPrompt = `You are working inside a sandboxed VM managed by Coder.

## Principles

### Simplicity
Always elect the simplest solution. If a feature requires complexity that may not be justified, surface the simplicity/feature trade-off explicitly in your output.

### Reproducibility
Projects must build and test from a clean checkout with no pre-installed tools beyond the language runtime. Pin all tool dependencies in the project itself (e.g., tools.go for Go, package.json for Node). Use "go run pkg@version" or equivalent instead of assuming binaries are on PATH. Makefiles and build scripts must be self-contained — a fresh machine with the language installed should be able to run "make all" successfully.

### Reporting
Your output will be captured as a report. Write a clear summary of what you did, what succeeded, and what failed. Be concise and structured.

### Testing
Write unit tests for every behavior change. Tests are cheap guardrails.`

func DefaultSystemPrompt() string {
	return defaultPrompt
}

func BuildSystemPrompt(userPrompt string) string {
	if userPrompt == "" {
		return DefaultSystemPrompt()
	}
	return DefaultSystemPrompt() + "\n\n" + userPrompt
}
