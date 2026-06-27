#!/usr/bin/env bash
# Conformance gate for the {{STANDARD_ID}} standard.
#
# Usage:  WATH_ARTIFACT_ROOT=/path/to/pr/worktree ./verify.sh
set -euo pipefail

ROOT="${WATH_ARTIFACT_ROOT:-$(pwd)}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "== Wath conformance :: {{STANDARD_ID}} =="
echo "artifact root: ${ROOT}"

echo "-- rule assertions"
WATH_ARTIFACT_ROOT="${ROOT}" pytest -q "${HERE}/test_conformance.py"

echo "== PASS =="
