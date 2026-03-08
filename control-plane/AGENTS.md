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

## Usage

### Prerequisites

Start PostgreSQL:

```bash
docker-compose up -d
```

### Build

```bash
make generate   # Generate API code from OpenAPI spec
make build      # Build binary to ./control-plane
```

Or build everything at once:

```bash
make all        # generate + build + test
```

### Run

```bash
DATABASE_URL="postgres://mother:mother@localhost:5432/control_plane" ./control-plane
```

The server listens on `:8080`. The only required configuration is the `DATABASE_URL` environment variable.

### Run (for agents)

Agents should start the control plane with output redirected to a known location:

```bash
mkdir -p /tmp/mother-control-plane
DATABASE_URL="postgres://mother:mother@localhost:5432/control_plane" \
  ./control-plane > /tmp/mother-control-plane/stdout.log 2> /tmp/mother-control-plane/stderr.log &
echo $! > /tmp/mother-control-plane/pid
```

- **Logs**: `/tmp/mother-control-plane/stdout.log`, `/tmp/mother-control-plane/stderr.log`
- **PID file**: `/tmp/mother-control-plane/pid`

To check if the control plane is already running:

```bash
if [ -f /tmp/mother-control-plane/pid ] && kill -0 "$(cat /tmp/mother-control-plane/pid)" 2>/dev/null; then
  echo "control plane is running (pid $(cat /tmp/mother-control-plane/pid))"
else
  echo "control plane is not running"
fi
```

To stop it:

```bash
kill "$(cat /tmp/mother-control-plane/pid)"
```

### API

The API is defined in `api/openapi.yaml`. That file is the authoritative reference for all endpoints, request/response schemas, and status codes.

#### Health check

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

#### Submit a coder job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "service": "coder",
    "params": {
      "project_dir": "/absolute/path/to/project",
      "prompt": "implement the feature described in spec.md",
      "model": "sonnet",
      "system_prompt": "optional system prompt",
      "env_vars": {"KEY": "value"}
    }
  }'
# {"id":"<uuid>"}  (202 Accepted)
```

Required fields: `service`, `params.project_dir`, `params.prompt`. Everything else is optional.

#### Poll job status

```bash
curl http://localhost:8080/api/v1/jobs/<uuid>
# {
#   "id": "<uuid>",
#   "service": "coder",
#   "status": "pending|running|completed|failed",
#   "params": {...},
#   "result": "string or null",
#   "error": "string or null",
#   "created_at": "2026-03-08T...",
#   "completed_at": "2026-03-08T... or null"
# }
```

### Tests

```bash
make test       # Runs go test -v -race ./...
```

### Make targets

| Target     | Description                              |
|------------|------------------------------------------|
| `all`      | generate + build + test                  |
| `generate` | Generate server code from OpenAPI spec   |
| `build`    | Build binary to `./control-plane`        |
| `test`     | Run all tests with race detector         |
| `clean`    | Remove build artifacts                   |

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
