"""
Conformance gate for the `{{STANDARD_ID}}` standard.

Each test is named for the rule it enforces (1:1 rule→test mapping).
Run via conformance/verify.sh.
"""

import os
from pathlib import Path

import pytest

ROOT = Path(os.environ.get("WATH_ARTIFACT_ROOT", ".")).resolve()
STANDARD_DIR = Path(__file__).resolve().parent.parent


def test_{{RULE_PREFIX}}_001_placeholder():
    """TODO: Implement first rule assertion."""
    params_path = ROOT / "integration.params.json"
    assert params_path.exists(), "integration.params.json missing — emit typed params first"
