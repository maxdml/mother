# GTM Platform Migration Design

## Goal

Make gtm-platform a mother service: coder can develop on it, and it runs as a container.

## Components

### 1. Git Submodule — `gtm-platform/` at repo root

- Points to `github.com/maxdml/gtm-platform`
- Not initialized by default — developers opt in via init script
- Coder accesses gtm-platform by mounting the submodule checkout into the VM

### 2. Init Script — `tools/init-gtm-platform.sh`

- Runs `git submodule update --init gtm-platform`
- Validates the checkout succeeded

### 3. Dockerfile — `gtm-platform/Dockerfile` (in the gtm-platform repo)

- Python 3.14 base image
- Installs dependencies via `uv`
- Copies source code
- No secrets baked in — expects env vars at runtime
- Exposes port 8000

### 4. Docker Compose — `docker-compose.gtm-platform.yaml` at mother root

- **gtm-platform-db**: Postgres 16 + pgvector extension (local dev only)
- **gtm-platform**: builds from submodule Dockerfile, depends on db service
- Reads secrets from `.env` file (or SOPS-decrypted env vars)
- In production, skip the db service and point DATABASE_URL at Supabase

### 5. Coder Integration — no changes needed

- gtm-platform is a normal project directory
- Coder mounts the submodule path into the VM as it would any project

## File Layout

```
mother/
├── gtm-platform/                    # git submodule (opt-in)
│   ├── Dockerfile                   # added in gtm-platform repo
│   ├── src/
│   └── ...
├── docker-compose.gtm-platform.yaml # dev compose stack
├── tools/
│   └── init-gtm-platform.sh        # submodule init script
└── ...
```

## Out of Scope

- No Go rewrites or deep integration with mother packages
- No secrets baked into Docker images
- No changes to mother's existing control plane or API
- No CI/CD changes (can be added later)
