#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SECRETS_FILE="${XDG_RUNTIME_DIR:-/tmp}/.gtm-platform-secrets.env"

docker compose -f "$REPO_ROOT/docker-compose.gtm-platform.yaml" down "$@"

rm -f "$SECRETS_FILE"

echo "gtm-platform stopped. Secrets file cleaned up."
