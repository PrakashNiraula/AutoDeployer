#!/usr/bin/env bash
# Validate docs/contract/examples against JSON Schemas.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$ROOT/../.." && pwd)"
VENV="${CONTRACT_VENV:-$REPO/.venv-contract}"

if [[ ! -x "$VENV/bin/python" ]]; then
  python3 -m venv "$VENV"
  "$VENV/bin/pip" install -q -r "$ROOT/requirements.txt"
fi

exec "$VENV/bin/python" "$ROOT/validate_examples.py"
