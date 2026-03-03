# AGENTS.md Design

## Decision

Single narrative AGENTS.md at repo root, targeting agents as primary audience but readable by humans.

## Audience

Agents working on or with coder. Must also be clear to human contributors.

## Structure

1. **Header** — what coder is and where it's heading
2. **Architecture** — two-file Go CLI, Lima VM lifecycle, mount strategy, execution flow, modes
3. **Building** — Makefile targets, dependencies
4. **Testing** — unit test expectations, integration test scope, Go conventions
5. **Principles** — simplicity, reports (stored in `coder/reports/`, passed to sub-agents), testing
6. **Vision** — from CLI passthrough to autonomous orchestrator with DBOS workflow integration

## Key Decisions

- **Approach:** Single file (Approach 1) over modular docs or machine-readable schema. Matches simplicity principle; can be split later.
- **Location:** Repo root (`AGENTS.md`). This will be a monorepo.
- **Reports directory:** `coder/reports/` on the host, not inside the Lima VM.
- **Sub-agent delegation:** Coder must pass its principles (especially reporting requirements) to sub-agents like Claude Code.
- **Conventions:** Minimal but with three firm baselines — simplicity, reports, testing.
