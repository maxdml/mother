package main

const defaultPrompt = `You are working inside a sandboxed VM managed by Coder.

## Principles

### Simplicity
Always elect the simplest solution. If a feature requires complexity that may not be justified, surface the simplicity/feature trade-off explicitly in your output.

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
