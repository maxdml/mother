# AGENTS.md

Coder is an autonomous coding agent that receives instructions via API and executes them inside sandboxed Lima VMs. Today it acts as a direct passthrough to Claude Code — receiving a prompt, spinning up a VM, and running Claude Code inside it. Over time, coder will evolve into an orchestrator that delegates tasks to sub-agents (e.g., Claude Code) and coordinates their work.

This document describes how to understand, use, and maintain coder. It is written for agents but should be clear to humans.

## Architecture

Coder is a Go CLI:

- **main.go** — CLI entry point. Parses flags (including `--id`), loads `.env`, resolves auth credentials, wires report writing, and calls into the Lima layer.
- **lima.go** — VM lifecycle. Generates Lima YAML config from a Go template, starts/stops VMs via `limactl`, runs Claude Code inside them, and captures output.
- **report.go** — Report generation. Defines the `Report` struct and `WriteReport` function. Reports are written as JSON to `coder/reports/<id>.json`.
- **principles.go** — Default system prompt with coder's principles (simplicity, reporting, testing). Always injected into sub-agents via `BuildSystemPrompt`.

The VM runs Ubuntu 24.04, installs Node.js + Claude Code on provision, and mounts:
- The project directory (writable) — the codebase being worked on
- `~/.claude` (read-only) — OAuth credentials for Claude Max

### Flow

1. User invokes `coder <project-dir> [options]`
2. Coder resolves the project path, loads env vars from `.env` + CLI flags
3. Stages auth credentials into a temp dir inside `~/.claude`
4. Generates a Lima YAML config and starts the VM
5. Injects default principles + any user system prompt into Claude Code
6. Runs Claude Code inside the VM via `limactl shell`, capturing stdout
7. Writes a JSON report to `coder/reports/<id>.json`
8. On exit (or signal), force-deletes the VM

### Modes

- **Interactive** (default) — terminal attached, user talks to Claude Code directly
- **Print** (`-p` / `--print`) — non-interactive, passes a prompt and exits with output
- **Prompt file** (`-f` / `--prompt-file`) — reads prompt from a file, combinable with `-p`
- **Invocation ID** (`--id`) — passed by the control plane to track invocations; used as the report filename

### Principle Injection

Coder always injects a default system prompt into Claude Code containing its core principles (simplicity, reporting, testing). If the user provides `--system-prompt`, it is appended after the defaults. This ensures every sub-agent operates under coder's principles regardless of how it is invoked.

## Building

```
make build    # produces ./coder binary
make clean    # removes the binary
```

Dependencies: Go 1.26+, Lima (`limactl`) installed on the host.

## Testing

Unit tests are cheap guardrails — use them extensively. Every new function or behavior change should have corresponding unit tests. Integration tests should verify VM lifecycle and Claude Code execution end-to-end.

```
go test ./...
```

Test files live alongside the code they test, following Go convention.

## Principles

### Simplicity

Always elect the simplest solution. If a feature requires complexity that may not be justified, surface the simplicity/feature trade-off in the output report rather than silently choosing the complex path.

### Reports

Every coder invocation produces a report readable by both humans and agents. Reports are stored in the `coder/reports/` directory (this is on the host, not inside the Lima VM). Reports will eventually be consumed by DBOS workflows, and each invocation will carry a unique invocation ID. Design output formats with machine parseability in mind without sacrificing human readability.

Coder is responsible for passing these principles to its sub-agents (e.g., Claude Code). When delegating work, coder must ensure sub-agents understand the reporting requirements and produce output that coder can incorporate into its own reports.

### Testing

Unit tests are cheap — use them liberally as guardrails for every behavior. Integration tests verify that the full VM lifecycle works: provisioning, mounting, running Claude Code, and cleanup. Both are required for any non-trivial change.

## Vision

Coder is evolving from a CLI passthrough into an autonomous orchestrator.

**Today:** Coder receives a prompt, spins up a sandboxed VM, runs Claude Code inside it, and returns the result.

**Next:** Coder receives instructions from an API, decomposes work into tasks, delegates each task to a sub-agent (e.g., Claude Code) running in its own sandbox, collects reports, and returns a consolidated result. Each invocation will be tracked by a unique ID within DBOS workflows.

The Lima VM sandbox remains the foundation — every sub-agent runs isolated, with only its assigned project directory mounted writable.
