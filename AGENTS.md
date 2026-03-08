# MOTHER

Mother is a system trusted by a human operator to manage their affairs. It receives requests from various clients, interprets them, and dispatches work to specialized services.

## System Architecture

Mother is a single Go monorepo (`github.com/maxdml/mother`) containing:

- **Control plane** (`cmd/mother/`) — HTTP server that receives requests and dispatches them to services
- **Services** (`internal/`) — shared libraries that perform specific tasks
- **CLI tools** (`cmd/coder/`) — command-line interfaces for direct service access
- **API** (`api/`) — OpenAPI spec, generated code, and HTTP handlers
- **Clients** — interfaces that let the operator interact with Mother

```
Clients → API Gateway (cmd/mother) → Services (internal/) → VM sandbox
                                   ↑
CLI (cmd/coder) ─────────────────────┘
```

## Project Layout

```
mother/
├── cmd/
│   ├── mother/     # Control plane server binary
│   └── coder/      # Coder CLI binary
├── api/            # OpenAPI spec, generated code, HTTP handlers
├── internal/
│   ├── vm/         # Lima VM lifecycle management
│   ├── coder/      # Coder engine, principles, reports
│   └── workflow/   # DBOS durable workflows, job manager
├── go.mod          # module github.com/maxdml/mother
├── Makefile        # Build, test, generate
└── docker-compose.yaml  # PostgreSQL for DBOS
```

## Building

```bash
make all        # generate + build + test
make build      # builds bin/mother and bin/coder
make generate   # regenerates API code from OpenAPI spec
make test       # runs all tests with race detector
```

Prerequisites: Go 1.26+, Lima (`limactl`), PostgreSQL (via docker-compose).

## Running

### Control Plane

```bash
docker-compose up -d   # Start PostgreSQL
DATABASE_URL="postgres://mother:mother@localhost:5432/control_plane" ./bin/mother
```

Listens on `:8080`. See `api/openapi.yaml` for the full API spec.

### Coder CLI

```bash
./bin/coder <project-dir> -p "implement the feature"
```

See `coder/AGENTS.md` for CLI options and modes.

## Services

### Coder

An autonomous coding agent that runs Claude Code inside sandboxed Lima VMs. Available as both:
- **Library** (`internal/coder/`) — imported directly by the control plane
- **CLI** (`cmd/coder/`) — standalone command-line tool

The control plane calls the coder engine as a Go function (no subprocess). Both the CLI and the server share the same `internal/coder` and `internal/vm` packages.

## Clients

### CLI (current)

The coder CLI (`cmd/coder/`) and the control plane API.

### Planned

- **Graphical UI** — located at `mother-ui/`
- **Gmail** — receive and respond to requests via email
- **Signal** — receive and respond to requests via Signal messages

## Conventions

- Each significant component has documentation describing its purpose and architecture
- OpenAPI spec (`api/openapi.yaml`) is the source of truth for the HTTP API
- All code generated from specs lives alongside the specs
- `internal/` packages are not importable outside this module
