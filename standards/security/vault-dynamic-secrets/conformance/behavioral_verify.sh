#!/usr/bin/env bash
# Behavioral gate entry point — JWT stand-in for workload identity -> dynamic DB secret.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${WATH_ARTIFACT_ROOT:-$(pwd)}"

export WATH_ARTIFACT_ROOT="${ROOT}"
export VAULT_ADDR="${VAULT_ADDR:-http://127.0.0.1:8200}"
export VAULT_TOKEN="${VAULT_TOKEN:-root}"

if ! command -v vault >/dev/null 2>&1; then
  echo "behavioral: vault CLI not found — install HashiCorp Vault or set PATH" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "behavioral: python3 not found" >&2
  exit 1
fi

# Ensure Python deps for behavioral gate
python3 - <<'PY' 2>/dev/null || pip3 install -q -r "${HERE}/requirements.txt"
import jwt, psycopg2  # noqa: F401
PY

exec python3 "${HERE}/behavioral_verify.py"
