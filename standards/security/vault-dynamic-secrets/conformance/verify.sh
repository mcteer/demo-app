#!/usr/bin/env bash
# Conformance gate for the vault-dynamic-secrets standard.
#
# One entry point, two callers:
#   - Tier-1: Wath's sandbox runs this against the agent's freshly generated PR.
#   - Tier-2: the shipped CI workflow (VDS-008) runs this in the team's real environment.
#
# Usage:
#   WATH_ARTIFACT_ROOT=/path/to/worktree ./verify.sh
#   WATH_BEHAVIORAL=1 WATH_MANAGE_SANDBOX=1 ./verify.sh   # full Tier-1 + sandbox
#   SKIP_TOOLCHAIN=1 ./verify.sh                          # pytest only
set -euo pipefail

ROOT="${WATH_ARTIFACT_ROOT:-$(pwd)}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EVIDENCE_DIR="${ROOT}/.wath"

echo "== Wath conformance :: vault-dynamic-secrets =="
echo "artifact root: ${ROOT}"

mkdir -p "${EVIDENCE_DIR}"

# Optional: start ephemeral Vault + Postgres
if [ "${WATH_MANAGE_SANDBOX:-0}" = "1" ]; then
  echo "-- starting sandbox (Vault dev + Postgres)"
  "${HERE}/start-sandbox.sh"
  # shellcheck source=/dev/null
  source "${WATH_SANDBOX_ENV:-/tmp/wath-sandbox.env}"
fi

# 1. Rule assertions (the standard, compiled to checks)
echo "-- VDS-000..008 rule assertions"
WATH_ARTIFACT_ROOT="${ROOT}" python3 -m pytest -q "${HERE}/test_conformance.py"

if [ "${SKIP_TOOLCHAIN:-0}" != "1" ]; then
  # 2. Native toolchain validators
  if command -v vault >/dev/null 2>&1; then
    echo "-- vault policy fmt (syntax + canonical form)"
    vault policy fmt "${ROOT}/vault/policy.hcl" >/dev/null
  else
    echo "-- vault policy fmt: SKIPPED (vault CLI not installed; set SKIP_TOOLCHAIN=1 to silence)"
    exit 1
  fi

  if [ -d "${ROOT}/k8s" ]; then
    if command -v kubeconform >/dev/null 2>&1; then
      echo "-- kubeconform (manifest schema validity, incl. VSO CRDs)"
      kubeconform -strict -summary \
        -schema-location default \
        -schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
        "${ROOT}/k8s/" || {
          echo "kubeconform: manifest validation failed (network required for VSO CRD schemas)" >&2
          exit 1
        }
    else
      echo "-- kubeconform: SKIPPED (not installed)"
      exit 1
    fi
  fi
else
  echo "-- toolchain validators skipped (SKIP_TOOLCHAIN=1)"
fi

# 3. Behavioral gate (Tier-1 only)
if [ "${WATH_BEHAVIORAL:-0}" = "1" ]; then
  echo "-- behavioral: signed-identity -> role -> policy -> dynamic secret"
  WATH_ARTIFACT_ROOT="${ROOT}" "${HERE}/behavioral_verify.sh"
fi

# Summary artifact for PR attachment
SUMMARY="${EVIDENCE_DIR}/verify-summary.json"
python3 - <<PY
import json, os
from datetime import datetime, timezone
from pathlib import Path

root = Path("${ROOT}")
summary = {
    "standard": "vault-dynamic-secrets",
    "artifact_root": str(root),
    "verified_at": datetime.now(timezone.utc).isoformat(),
    "static_gate": "pass",
    "behavioral_gate": os.environ.get("WATH_BEHAVIORAL", "0") == "1",
}
evidence = root / ".wath" / "verification-evidence.json"
if evidence.exists():
    summary["behavioral_evidence"] = json.loads(evidence.read_text())
(root / ".wath" / "verify-summary.json").write_text(json.dumps(summary, indent=2) + "\\n")
print(f"summary -> {root / '.wath' / 'verify-summary.json'}")
PY

echo "== PASS =="
