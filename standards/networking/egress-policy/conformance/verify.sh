#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STANDARD_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ARTIFACT_ROOT="${WATH_ARTIFACT_ROOT:-$(pwd)}"

export WATH_ARTIFACT_ROOT="${ARTIFACT_ROOT}"
export PYTHONPATH="${STANDARD_ROOT}/conformance:${PYTHONPATH:-}"

echo "== egress-policy verify =="
echo "artifact root: ${ARTIFACT_ROOT}"

python3 -m pytest "${SCRIPT_DIR}/test_conformance.py" -q

mkdir -p "${ARTIFACT_ROOT}/.wath"
echo '{"standard":"egress-policy","passed":true}' > "${ARTIFACT_ROOT}/.wath/verify-summary.json"
echo "== PASS =="
