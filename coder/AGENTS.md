# AGENTS.md

Coder is an autonomous coding agent that receives instructions via API and executes them inside sandboxed Lima VMs. Today it acts as a direct passthrough to Claude Code — receiving a prompt, spinning up a VM, and running Claude Code inside it. Over time, coder will evolve into an orchestrator that delegates tasks to sub-agents (e.g., Claude Code) and coordinates their work.

This document describes how to understand, use, and maintain coder. It is written for agents but should be clear to humans.

## Architecture

Coder is a Go CLI with two files:

- **main.go** — CLI entry point. Parses flags, loads `.env`, resolves auth credentials, and calls into the Lima layer.
- **lima.go** — VM lifecycle. Generates Lima YAML config from a Go template, starts/stops VMs via `limactl`, and runs Claude Code inside them.

The VM runs Ubuntu 24.04, installs Node.js + Claude Code on provision, and mounts:
- The project directory (writable) — the codebase being worked on
- `~/.claude` (read-only) — OAuth credentials for Claude Max

### Flow

1. User invokes `coder <project-dir> [options]`
2. Coder resolves the project path, loads env vars from `.env` + CLI flags
3. Stages auth credentials into a temp dir inside `~/.claude`
4. Generates a Lima YAML config and starts the VM
5. Runs Claude Code inside the VM via `limactl shell`
6. On exit (or signal), force-deletes the VM

### Modes

- **Interactive** (default) — terminal attached, user talks to Claude Code directly
- **Print** (`-p` / `--print`) — non-interactive, passes a prompt and exits with output
- **Prompt file** (`-f` / `--prompt-file`) — reads prompt from a file, combinable with `-p`

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
