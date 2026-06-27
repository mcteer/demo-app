"""
Conformance gate for the `egress-policy` standard.

Each test is named for the rule it enforces (1:1 rule→test mapping).
Run via conformance/verify.sh with WATH_ARTIFACT_ROOT set to the consumer worktree.
"""

import json
import os
from pathlib import Path

import jsonschema
import pytest
import yaml

ROOT = Path(os.environ.get("WATH_ARTIFACT_ROOT", ".")).resolve()
STANDARD_DIR = Path(__file__).resolve().parent.parent
SCHEMA = json.loads((STANDARD_DIR / "schema" / "integration.params.schema.json").read_text())


def _load_params():
    p = ROOT / "integration.params.json"
    assert p.exists(), "integration.params.json missing"
    data = json.loads(p.read_text())
    jsonschema.validate(data, SCHEMA)
    return data


def test_EGR_001_explicit_destinations_no_wildcards():
    params = _load_params()
    for rule in params["egress"]:
        assert "*" not in rule["host"], f"wildcard host: {rule['host']}"


def test_EGR_002_no_broad_internet_egress():
    params = _load_params()
    for rule in params["egress"]:
        host = rule["host"]
        assert host not in ("0.0.0.0", "0.0.0.0/0", "*"), f"broad egress: {host}"


def test_artifact_networkpolicy_when_kubernetes():
    params = _load_params()
    if params["runtime"] == "kubernetes":
        np = ROOT / "k8s" / "networkpolicy-egress.yaml"
        assert np.exists(), "k8s/networkpolicy-egress.yaml required for kubernetes runtime"
        yaml.safe_load(np.read_text())
