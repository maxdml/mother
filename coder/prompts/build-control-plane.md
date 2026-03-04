Build the Mother control plane as described in AGENTS.md.

Read AGENTS.md and api/openapi.yaml first — they are the source of truth.

## What to build

A Go HTTP server that:

1. Serves the API defined in api/openapi.yaml on localhost:8080
2. Uses oapi-codegen to generate server interfaces and types from the OpenAPI spec
3. Implements the async job model: POST /api/v1/jobs creates a job, GET /api/v1/jobs/{id} retrieves status/result
4. Implements the /health endpoint
5. Shells out to the coder binary at /Users/mother/mother/coder/coder to run jobs
6. Uses DBOS for workflow orchestration (or if DBOS Go SDK is not available, implement a simple in-memory job store as a stepping stone — note this in output)

## Project structure

- go.mod (module: mother/control-plane)
- main.go — server entry point, starts HTTP server on :8080
- api/openapi.yaml — already exists, do not modify
- api/generate.go — go:generate directive for oapi-codegen
- api/server.gen.go — generated server interface and types
- handler.go — implements the generated server interface
- service.go — coder service: builds command, executes coder binary, captures output
- job.go — job store (manages job state: pending/running/completed/failed)
- Makefile — build, generate, test targets

## Key requirements

- Install oapi-codegen and run code generation as the first step
- The coder binary path is /Users/mother/mother/coder/coder
- Pass --id (job UUID) to coder so reports are tracked
- Pass -p with the prompt from CoderParams
- Pass other CoderParams (env_vars, system_prompt, model) as appropriate CLI flags
- Jobs run asynchronously in goroutines
- Job state transitions: pending -> running -> completed/failed
- Return 202 with job ID on POST, return full Job object on GET
- Health endpoint returns {"status": "ok"}

## Testing

Write comprehensive tests:
- Unit tests for the job store
- Unit tests for the service layer (mock the coder binary execution)
- Handler tests using httptest
- Verify the server compiles and tests pass

## Output

Summarize what you built, what works, and any decisions or tradeoffs you made.
