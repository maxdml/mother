# MOTHER's CONTROL PLANE

The control plane is Mother's central API server. It receives requests from clients and dispatches them to services. It is a long-lived HTTP server written in Go, listening on localhost.

Coder is the first service. More services will be added over time. The architecture must support adding new services without restructuring.

## Architecture

- **Language**: Go
- **Database**: PostgreSQL (used by DBOS for workflow state)
- **Workflows**: DBOS handles workflow orchestration, durability, and retries
- **API style**: REST, spec-first. The OpenAPI spec at `api/openapi.yaml` is the source of truth. Server code is generated from it using `oapi-codegen`.
- **Pattern**: Handler (generated interface) -> Workflow (DBOS) -> Service (business logic) -> Clients (interfaces to interact with external systems)
- **Auth**: None. Localhost only.

### Request flow

1. Client sends HTTP request
2. Generated handler routes to the implementation
3. Handler delegates to a DBOS workflow
4. Workflow calls the service layer
5. Service executes the operation (e.g., shells out to coder binary)
6. Workflow persists state, handler returns response

### Async job model

Long-running operations (like coder) use an async pattern:
- **POST** creates the job and returns a job ID immediately
- **GET** retrieves the job status and result by ID
- Job state is managed by DBOS workflows

## API

The API is defined in `api/openapi.yaml`. That file is the authoritative reference for all endpoints, request/response schemas, and status codes.

## Services

### Coder

Coder is invoked by shelling out to the coder binary at `/Users/mother/mother/coder/coder`. The control plane passes arguments matching the coder CLI, discoverable in /Users/mother/mother/coder/AGENTS.md

The service layer is responsible for constructing the command, executing it, and capturing output.

## Testing

Thorough unit, integration, and contract testing is required for every component of the system. Contract tests must validate that the server implementation conforms to the OpenAPI spec.

## Acceptance Criteria

- The server starts, connects to Postgres, and serves the API on localhost
- A client can submit a coder job and retrieve its status and result
- The OpenAPI spec is accurate and complete
- All tests pass
