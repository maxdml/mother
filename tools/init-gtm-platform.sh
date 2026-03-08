#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "Initializing gtm-platform submodule..."
cd "$REPO_ROOT"
git submodule update --init gtm-platform

if [ ! -f gtm-platform/pyproject.toml ]; then
    echo "ERROR: submodule checkout appears incomplete (missing pyproject.toml)" >&2
    exit 1
fi

echo "gtm-platform submodule initialized successfully."
