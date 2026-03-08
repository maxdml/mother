#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SECRETS_FILE="${XDG_RUNTIME_DIR:-/tmp}/.gtm-platform-secrets.env"

# Decrypt gtm-platform secrets from SOPS (global + service-specific)
sops -d --output-type json ~/.mother/secrets.yaml \
  | python3 -c "
import sys, json
d = json.load(sys.stdin)
merged = {**d.get('global', {}), **d.get('gtm-platform', {})}
for k, v in merged.items():
    print(f'{k}={v}')
" > "$SECRETS_FILE"

chmod 600 "$SECRETS_FILE"

# Start the compose stack, mounting the decrypted secrets into the container
GTM_SECRETS_FILE="$SECRETS_FILE" \
  docker compose -f "$REPO_ROOT/docker-compose.gtm-platform.yaml" "$@" up -d

echo "gtm-platform started. Secrets mounted from $SECRETS_FILE"
echo "Run 'tools/stop-gtm-platform.sh' to stop and clean up secrets."
