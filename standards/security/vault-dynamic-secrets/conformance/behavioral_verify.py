#!/usr/bin/env python3
"""
Behavioral gate: prove signed-identity -> role -> policy -> dynamic database secret.

Uses JWT auth as a stand-in for kubernetes workload identity (no cluster required).
Writes structured evidence to ${WATH_ARTIFACT_ROOT}/.wath/verification-evidence.json
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path

import jwt
import psycopg2
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa

ROOT = Path(os.environ["WATH_ARTIFACT_ROOT"]).resolve()
HERE = Path(__file__).resolve().parent
VAULT_ADDR = os.environ.get("VAULT_ADDR", "http://127.0.0.1:8200")
VAULT_TOKEN = os.environ.get("VAULT_TOKEN", "root")
POSTGRES_ADMIN_URL = os.environ.get(
    "WATH_POSTGRES_ADMIN_URL",
    "postgresql://orders:orders@127.0.0.1:5432/orders?sslmode=disable",
)


def vault(*args: str, token: str | None = None) -> str:
    env = {**os.environ, "VAULT_ADDR": VAULT_ADDR}
    if token:
        env["VAULT_TOKEN"] = token
    result = subprocess.run(
        ["vault", *args],
        capture_output=True,
        text=True,
        env=env,
        check=True,
    )
    return result.stdout


def load_params() -> dict:
    return json.loads((ROOT / "integration.params.json").read_text())


def ensure_postgres_schema() -> None:
    ddl = """
    CREATE SCHEMA IF NOT EXISTS orders;
    CREATE TABLE IF NOT EXISTS orders.orders (
        id SERIAL PRIMARY KEY,
        customer TEXT NOT NULL,
        amount NUMERIC(10, 2) NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    """
    with psycopg2.connect(POSTGRES_ADMIN_URL) as conn:
        conn.autocommit = True
        with conn.cursor() as cur:
            cur.execute(ddl)


def configure_vault(params: dict, policy_hcl: str) -> tuple[str, str, str]:
    """Configure database + jwt auth; return (jwt_role_name, bound_subject, private_pem)."""
    mounts = json.loads(vault("secrets", "list", "-format=json", token=VAULT_TOKEN))
    if "database/" not in mounts:
        vault("secrets", "enable", "-path=database", "database", token=VAULT_TOKEN)

    conn_url = POSTGRES_ADMIN_URL.replace("postgresql://", "postgres://")
    vault(
        "write",
        "database/config/orders-api",
        f"plugin_name=postgresql-database-plugin",
        f"allowed_roles={params['db_role']}",
        f"connection_url={conn_url}",
        token=VAULT_TOKEN,
    )

    creation = (
        'CREATE ROLE "{{name}}" WITH LOGIN PASSWORD \'{{password}}\' VALID UNTIL \'{{expiration}}\'; '
        'GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA orders TO "{{name}}"; '
        'GRANT USAGE ON SCHEMA orders TO "{{name}}";'
    )
    vault(
        "write",
        f"database/roles/{params['db_role']}",
        f"db_name=orders-api",
        f"creation_statements={creation}",
        f"default_ttl={params['default_ttl_seconds']}s",
        f"max_ttl={params['max_ttl_seconds']}s",
        token=VAULT_TOKEN,
    )

    policy_name = params["identity_binding"]["policy_name"]
    with tempfile.NamedTemporaryFile("w", suffix=".hcl", delete=False) as fh:
        fh.write(policy_hcl)
        policy_file = fh.name
    try:
        vault("policy", "write", policy_name, policy_file, token=VAULT_TOKEN)
    finally:
        os.unlink(policy_file)

    auths = json.loads(vault("auth", "list", "-format=json", token=VAULT_TOKEN))
    if "jwt/" not in auths:
        vault("auth", "enable", "jwt", token=VAULT_TOKEN)

    private_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    public_pem = private_key.public_key().public_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PublicFormat.SubjectPublicKeyInfo,
    ).decode()

    with tempfile.NamedTemporaryFile("w", suffix=".pem", delete=False) as fh:
        fh.write(public_pem)
        pub_file = fh.name
    try:
        vault(
            "write",
            "auth/jwt/config",
            f"jwt_validation_pubkeys=@{pub_file}",
            token=VAULT_TOKEN,
        )
    finally:
        os.unlink(pub_file)

    ns = params["identity_binding"]["k8s_namespaces"][0]
    sa = params["identity_binding"]["k8s_service_accounts"][0]
    bound_subject = f"system:serviceaccount:{ns}:{sa}"
    jwt_role = params["app_name"]

    vault(
        "write",
        f"auth/jwt/role/{jwt_role}",
        "role_type=jwt",
        "bound_audiences=vault",
        "user_claim=sub",
        f"bound_subject={bound_subject}",
        f"token_policies={policy_name}",
        token=VAULT_TOKEN,
    )

    private_pem = private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )
    return jwt_role, bound_subject, private_pem.decode()


def mint_jwt(bound_subject: str, private_pem: str) -> str:
    now = int(time.time())
    claims = {
        "sub": bound_subject,
        "aud": "vault",
        "iat": now,
        "exp": now + 300,
    }
    return jwt.encode(claims, private_pem, algorithm="RS256")


def login_and_read_creds(jwt_role: str, jwt_token: str, creds_path: str) -> dict:
    out = vault(
        "write",
        "-format=json",
        f"auth/jwt/login",
        f"role={jwt_role}",
        f"jwt={jwt_token}",
    )
    login = json.loads(out)
    client_token = login["auth"]["client_token"]

    creds_out = vault("read", "-format=json", creds_path, token=client_token)
    return json.loads(creds_out)


def verify_postgres_login(username: str, password: str) -> int:
    admin = POSTGRES_ADMIN_URL.replace("postgresql://", "postgres://")
    host_part = admin.split("@", 1)[1]
    dsn = f"postgresql://{username}:{password}@{host_part}"
    with psycopg2.connect(dsn) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT COUNT(*) FROM orders.orders")
            row = cur.fetchone()
            return int(row[0]) if row else 0


def write_evidence(payload: dict) -> Path:
    out_dir = ROOT / ".wath"
    out_dir.mkdir(parents=True, exist_ok=True)
    path = out_dir / "verification-evidence.json"
    path.write_text(json.dumps(payload, indent=2) + "\n")
    return path


def main() -> int:
    params = load_params()
    policy_hcl = (ROOT / "vault" / "policy.hcl").read_text()
    creds_path = params["creds_path"]

    print(f"behavioral: postgres admin -> {POSTGRES_ADMIN_URL.split('@')[-1]}")
    ensure_postgres_schema()

    print("behavioral: configuring ephemeral Vault (database + jwt stand-in)")
    jwt_role, bound_subject, private_pem = configure_vault(params, policy_hcl)

    jwt_token = mint_jwt(bound_subject, private_pem)
    print(f"behavioral: jwt login as {bound_subject!r}")
    creds_payload = login_and_read_creds(jwt_role, jwt_token, creds_path)

    data = creds_payload["data"]
    username = data["username"]
    password = data["password"]
    lease_duration = creds_payload.get("lease_duration")
    order_count = verify_postgres_login(username, password)

    max_ttl = params["max_ttl_seconds"]
    if lease_duration is None or lease_duration > max_ttl:
        raise SystemExit(
            f"lease_duration {lease_duration} exceeds max_ttl {max_ttl}"
        )

    evidence = {
        "standard": "vault-dynamic-secrets",
        "verified_at": datetime.now(timezone.utc).isoformat(),
        "auth_stand_in": "jwt",
        "bound_subject": bound_subject,
        "creds_path": creds_path,
        "lease_duration_seconds": lease_duration,
        "max_ttl_seconds": max_ttl,
        "postgres_query": "SELECT COUNT(*) FROM orders.orders",
        "order_count": order_count,
        "result": "pass",
    }
    out = write_evidence(evidence)
    print(f"behavioral: PASS — dynamic cred issued, lease={lease_duration}s, orders={order_count}")
    print(f"behavioral: evidence -> {out}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except subprocess.CalledProcessError as exc:
        print(exc.stderr or exc.stdout or str(exc), file=sys.stderr)
        raise SystemExit(1) from exc
