# Control Plane Design

Date: 2026-03-02

## Context

Mother is a system composed of a control plane and services. The control plane is the central API server. Coder is the first service — we're bootstrapping the control plane with coder itself.

## Decisions

- **Runtime**: Long-lived HTTP server on localhost, no auth
- **Language**: Go
- **Database**: PostgreSQL (DBOS state only, no custom tables)
- **Workflows**: DBOS from day one
- **API**: REST, spec-first with OpenAPI + oapi-codegen
- **Coder integration**: Shell out to coder binary (library import later)
- **Async model**: POST returns job ID, GET polls for status/result
- **Testing**: Thorough unit, integration, and contract tests required. Coder chooses tooling and test design.
- **Client**: CLI only for v1

## Deliverables

- `AGENTS.md` — complete build instructions for coder
- `api/openapi.yaml` — authoritative API spec

## What's deferred

- Authentication/authorization
- Additional clients (UI, Signal, Gmail)
- Custom database tables beyond DBOS
- Coder as imported Go library (currently subprocess)
- Job history/service registry
