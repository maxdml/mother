# MOTHER

Mother is a system trusted by a human operator to manage their affairs. It receives requests from various clients, interprets them, and dispatches work to specialized services.

## System Architecture

Mother is composed of:

- **Control plane** — a central API server that receives requests and dispatches them to services
- **Services** — independent components that perform specific tasks (coding, and more over time)
- **Clients** — interfaces that let the operator interact with Mother

```
Clients → Control Plane → Services → Clients
```

## Control Plane

A long-lived HTTP server that exposes Mother's APIs. Some APIs are direct passthroughs to services. Others use AI to interpret natural language input and route to the appropriate service.

Located at `/Users/mother/mother/control-plane`. See its `AGENTS.md` for architecture, API spec, and implementation details.

## Services

### Coder

A coding service that takes a prompt and a project directory, then uses Claude Code in a sandboxed Lima VM to create or modify code. Coder is how Mother bootstraps itself — it builds and evolves all other components.

Located at `/Users/mother/mother/coder`. See its `AGENTS.md` for more information.

## Clients

### CLI (current)

The control plane can be managed with a CLI that talks to the HTTP API.

### Planned

- **Graphical UI** — located at `/Users/mother/mother/mother-ui`
- **Gmail** — receive and respond to requests via email
- **Signal** — receive and respond to requests via Signal messages

## Conventions

- Each component has its own `AGENTS.md` describing its purpose, architecture, and acceptance criteria
