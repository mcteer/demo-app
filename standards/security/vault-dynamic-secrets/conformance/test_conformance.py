"""
Conformance gate for the `vault-dynamic-secrets` standard.

This is the deterministic half of the standard. It is COMPILED FROM THE SAME RULES the
agent was steered with (SKILL.md §5), but it executes independently of the model: it reads
the artifacts the agent produced and asserts VDS-001..008 with ordinary code. The model
cannot talk its way past it.

Each test is named for the rule it enforces, so the mapping from prose rule -> executable
check is 1:1 and auditable. This same suite runs in Wath's Tier-1 sandbox (against the
agent's freshly generated PR) and is re-run by the shipped CI workflow in the team's own
environment (Tier-2).

Run:  pytest -q   (see verify.sh, which also runs `vault policy fmt` and `kubeconform`)
Deps: pytest, jsonschema, python-hcl2, pyyaml
Artifacts are resolved relative to $WATH_ARTIFACT_ROOT (default: current dir).
"""

import json
import os
import re
from pathlib import Path

import hcl2
import pytest
import yaml
from jsonschema import validate as jsonschema_validate

ROOT = Path(os.environ.get("WATH_ARTIFACT_ROOT", ".")).resolve()
STANDARD_DIR = Path(__file__).resolve().parent.parent

# Static connection strings / embedded creds we must never see at the consumer (VDS-001/007).
STATIC_CRED_PATTERNS = [
    re.compile(r"(postgres|postgresql|mysql)://[^/\s:]+:[^@/\s]+@", re.I),  # user:pass@host
    re.compile(r"\bpassword\s*[:=]\s*['\"]?[^\s'\"]{6,}", re.I),
    re.compile(r"secret/data/.*(db|database|postgres|mysql)", re.I),        # KV mount for DB creds
]
# Directories outside consumer integration artifacts (VDS-001 repo scan).
# `wath` — CI checks out the Wath standards repo under the workspace for verify.sh.
# `standards` — vendored standard tree (not app-owned integration files).
SCAN_EXCLUDE_DIRS = {
    "vault",
    "conformance",
    "schema",
    "standards",
    "wath",
    ".git",
    "node_modules",
    ".wath",
}


def _scan_exclude_dirs() -> set[str]:
    extra = os.environ.get("WATH_SCAN_EXCLUDE", "")
    extras = {d.strip() for d in extra.split(",") if d.strip()}
    return SCAN_EXCLUDE_DIRS | extras


# ---------- loaders ----------

def _load_params():
    p = ROOT / "integration.params.json"
    assert p.exists(), "integration.params.json missing — agent must emit typed params first (SKILL.md §3)"
    return json.loads(p.read_text())


def _load_policy_paths():
    """Return {path_string: {capabilities: [...]}} from the generated policy.hcl."""
    p = ROOT / "vault" / "policy.hcl"
    assert p.exists(), "vault/policy.hcl missing"
    with p.open() as fh:
        doc = hcl2.load(fh)
    paths = {}
    for block in doc.get("path", []):
        for path_str, body in block.items():
            key = path_str.strip('"')
            caps = body.get("capabilities", [])
            if caps and isinstance(caps[0], str):
                caps = [c.strip('"') for c in caps]
            paths[key] = {**body, "capabilities": caps}
    assert paths, "policy.hcl declares no path blocks"
    return paths


def _load_manifests():
    docs = []
    for f in (ROOT / "k8s").glob("**/*.y*ml") if (ROOT / "k8s").exists() else []:
        docs.extend(d for d in yaml.safe_load_all(f.read_text()) if d)
    return docs


def _consumer_files():
    exclude = _scan_exclude_dirs()
    for f in ROOT.rglob("*"):
        if f.is_file() and not (set(f.relative_to(ROOT).parts) & exclude):
            if "PULL_REQUEST_TEMPLATE" in f.relative_to(ROOT).parts:
                continue
            yield f


@pytest.fixture(scope="module")
def params():
    return _load_params()


# ---------- VDS-000: typed params are schema-valid (layer 2 gate) ----------

def test_VDS_000_params_schema_valid(params):
    schema = json.loads((STANDARD_DIR / "schema" / "integration.params.schema.json").read_text())
    jsonschema_validate(instance=params, schema=schema)  # raises on violation


# ---------- VDS-001: dynamic, not static ----------

def test_VDS_001_uses_dynamic_creds_path(params):
    assert params["creds_path"].startswith("database/creds/"), \
        "must consume a database/creds/<role> dynamic path, not a KV mount"


def test_VDS_001_no_static_credentials_in_repo():
    offenders = []
    for f in _consumer_files():
        try:
            text = f.read_text(errors="ignore")
        except Exception:
            continue
        for pat in STATIC_CRED_PATTERNS:
            if pat.search(text):
                offenders.append(f"{f.relative_to(ROOT)} :: /{pat.pattern[:40]}.../")
    assert not offenders, "static credential / KV-for-DB pattern found:\n" + "\n".join(offenders)


# ---------- VDS-002: TTL ceilings ----------

def test_VDS_002_ttl_ceilings(params):
    assert params["default_ttl_seconds"] <= 1800, "default_ttl exceeds 30m ceiling"
    assert params["max_ttl_seconds"] <= 3600, "max_ttl exceeds 1h ceiling"
    assert params["default_ttl_seconds"] <= params["max_ttl_seconds"], "default_ttl > max_ttl"


# ---------- VDS-003: read-only creds path ----------

def test_VDS_003_creds_path_is_read_only(params):
    paths = _load_policy_paths()
    creds_path = params["creds_path"]
    assert creds_path in paths, f"policy does not grant the declared creds_path {creds_path!r}"
    caps = paths[creds_path].get("capabilities", [])
    assert caps == ["read"], f"creds path capabilities must be exactly ['read'], got {caps}"


# ---------- VDS-004: no wildcard grants ----------

def test_VDS_004_no_wildcard_paths():
    bad = [p for p in _load_policy_paths() if "*" in p]
    assert not bad, f"policy contains wildcard path grants: {bad}"


# ---------- VDS-005: runtime-correct auth ----------

RUNTIME_AUTH = {
    "kubernetes": {"kubernetes"},
    "nomad": {"jwt"},
    "vm": {"approle", "aws", "gcp", "azure"},
}

def test_VDS_005_auth_method_matches_runtime(params):
    allowed = RUNTIME_AUTH[params["runtime"]]
    assert params["auth_method"] in allowed, \
        f"runtime {params['runtime']!r} requires auth_method in {sorted(allowed)}, got {params['auth_method']!r}"


# ---------- VDS-006: bound identity + scoped policy ----------

def test_VDS_006_policy_is_scoped_not_default(params):
    name = params["identity_binding"]["policy_name"]
    assert name != "default" and "admin" not in name, f"auth role must attach a scoped policy, not {name!r}"


def test_VDS_006_identity_binding_is_specific(params):
    b = params["identity_binding"]
    if params["runtime"] == "kubernetes":
        sas = b.get("k8s_service_accounts", [])
        ns = b.get("k8s_namespaces", [])
        assert sas and "*" not in sas, "k8s auth role must bind specific service accounts (no '*')"
        assert ns and "*" not in ns, "k8s auth role must bind specific namespaces (no '*')"
    elif params["runtime"] == "nomad":
        assert b.get("bound_claims"), "nomad jwt auth must bind workload-identity claims"


# ---------- VDS-007: no plaintext at the consumer ----------

def test_VDS_007_secret_delivery_present(params):
    docs = _load_manifests()
    kinds = {d.get("kind") for d in docs}
    if params["secret_delivery"] == "vso":
        assert "VaultDynamicSecret" in kinds, "VSO selected but no VaultDynamicSecret CR rendered"
    else:  # agent-injector
        has_annotations = any(
            any(k.startswith("vault.hashicorp.com/") for k in
                (d.get("spec", {}).get("template", {}).get("metadata", {}).get("annotations", {}) or {}))
            for d in docs if d.get("kind") == "Deployment"
        )
        assert has_annotations, "agent-injector selected but no vault.hashicorp.com/* annotations found"


def test_VDS_007_no_plaintext_secret_objects():
    docs = _load_manifests()
    for d in docs:
        if d.get("kind") == "Secret":
            blob = json.dumps(d.get("stringData", {})) + json.dumps(d.get("data", {}))
            assert not re.search(r"(password|postgres|mysql|user:)", blob, re.I), \
                "a Secret manifest carries DB credential material — must be issued dynamically, not committed"


# ---------- VDS-008: durable verification ----------

def test_VDS_008_ships_ci_verification():
    candidates = list((ROOT / ".github" / "workflows").glob("*vault*verif*.y*ml")) \
        if (ROOT / ".github" / "workflows").exists() else []
    assert candidates, "integration must ship a CI workflow that re-verifies the dynamic-secret flow"
