#!/usr/bin/env bash
# Start ephemeral Vault (dev mode) + Postgres for Tier-1 sandbox verification.
# Writes connection env to ${WATH_SANDBOX_ENV:-/tmp/wath-sandbox.env}
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
ENV_FILE="${WATH_SANDBOX_ENV:-/tmp/wath-sandbox.env}"
PGDATA="${WATH_PGDATA:-/tmp/wath-pgdata}"
VAULT_LOG="${WATH_VAULT_LOG:-/tmp/wath-vault.log}"

export VAULT_ADDR="${VAULT_ADDR:-http://127.0.0.1:8200}"
export VAULT_TOKEN="${VAULT_TOKEN:-root}"
export WATH_POSTGRES_ADMIN_URL="${WATH_POSTGRES_ADMIN_URL:-postgresql://orders:orders@127.0.0.1:5432/orders?sslmode=disable}"

start_postgres() {
  if command -v pg_isready >/dev/null 2>&1 && pg_isready -h 127.0.0.1 -p 5432 -U orders -d orders 2>/dev/null; then
    echo "sandbox: postgres already reachable on :5432"
    return 0
  fi

  if command -v docker >/dev/null 2>&1; then
    echo "sandbox: starting postgres via docker compose"
    docker compose -f "${REPO_ROOT}/examples/consumer-demo/docker-compose.yml" up -d postgres
    for _ in $(seq 1 30); do
      if docker compose -f "${REPO_ROOT}/examples/consumer-demo/docker-compose.yml" exec -T postgres pg_isready -U orders -d orders 2>/dev/null; then
        return 0
      fi
      sleep 1
    done
    echo "sandbox: postgres docker compose failed to become ready" >&2
    exit 1
  fi

  if command -v initdb >/dev/null 2>&1 && command -v pg_ctl >/dev/null 2>&1; then
    if [ ! -d "${PGDATA}/PG_VERSION" ]; then
      echo "sandbox: initializing local postgres at ${PGDATA}"
      initdb -D "${PGDATA}" -U orders --auth=trust >/dev/null
      echo "listen_addresses = '127.0.0.1'" >> "${PGDATA}/postgresql.conf"
      echo "port = 5432" >> "${PGDATA}/postgresql.conf"
    fi
    pg_ctl -D "${PGDATA}" -l /tmp/wath-pg.log start -w
    createdb -h 127.0.0.1 -U orders orders 2>/dev/null || true
    psql -h 127.0.0.1 -U orders -d orders -f "${REPO_ROOT}/examples/consumer-demo/db/init.sql" >/dev/null
    return 0
  fi

  echo "sandbox: no postgres available (need :5432, docker, or initdb/pg_ctl)" >&2
  exit 1
}

start_vault() {
  if vault status >/dev/null 2>&1; then
    echo "sandbox: vault already reachable at ${VAULT_ADDR}"
    return 0
  fi

  if ! command -v vault >/dev/null 2>&1; then
    echo "sandbox: vault CLI not found" >&2
    exit 1
  fi

  echo "sandbox: starting vault dev server -> ${VAULT_LOG}"
  nohup vault server -dev \
    -dev-root-token-id="${VAULT_TOKEN}" \
    -dev-listen-address=127.0.0.1:8200 \
    >"${VAULT_LOG}" 2>&1 &
  echo $! > /tmp/wath-vault.pid

  for _ in $(seq 1 20); do
    if vault status >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "sandbox: vault dev server failed to start" >&2
  exit 1
}

start_postgres
start_vault

cat >"${ENV_FILE}" <<EOF
export VAULT_ADDR="${VAULT_ADDR}"
export VAULT_TOKEN="${VAULT_TOKEN}"
export WATH_POSTGRES_ADMIN_URL="${WATH_POSTGRES_ADMIN_URL}"
EOF
echo "sandbox: env written to ${ENV_FILE}"
