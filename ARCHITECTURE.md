# catalog-service — application architecture

> **For maintainers:** this service uses **tier-4 Vault dynamic database secrets** for
> Postgres credentials. See `integration.params.json`, `vault/`, and `k8s/` for the
> Wath onboarding artifacts.

---

## 1. What the app is

`catalog-service` is a small HTTP API over a Postgres `products` table.

**Behaviour:**
- `GET /healthz` — liveness. Returns `200 {"status":"ok"}`. Must **not** touch the database.
- `GET /readyz` — readiness. Executes `SELECT 1` against Postgres; `200` if it succeeds, `503`
  otherwise.
- `GET /products` — returns all rows from `products` as JSON.
- `GET /products/{id}` — returns one row, or `404`.

---

## 2. Stack

- **Language:** Go 1.22+
- **Postgres driver:** `jackc/pgx` pool
- **HTTP:** standard library `net/http` with `chi`
- **Config:** environment variables only (see §4)
- **Container:** multi-stage `Dockerfile`

---

## 3. Repository layout

```
catalog-service/
  cmd/server/main.go
  internal/config/config.go       # THE CREDENTIAL SEAM
  internal/db/db.go
  internal/handlers/handlers.go
  internal/store/store.go
  migrations/0001_init.sql
  k8s/                            # tier-4 manifests (VSO + Deployment)
  vault/                          # policy + auth role (admin applies)
  integration.params.json
  Dockerfile
  docker-compose.yaml             # local dev only — creds via .env
  README.md
```

---

## 4. Configuration contract (the credential seam)

All database configuration is read **in one place** — `internal/config/config.go` — from these
environment variables:

```
DB_HOST       e.g. postgres.catalog.svc.cluster.local
DB_PORT       e.g. 5432
DB_NAME       e.g. catalog
DB_USER       e.g. (VSO-synced username)
DB_PASSWORD   e.g. (VSO-synced password)
DB_SSLMODE    e.g. disable (demo) / require (prod)
PORT          HTTP listen port, default 8080
```

In Kubernetes, `DB_USER` and `DB_PASSWORD` are sourced from the VSO-managed Secret
`catalog-service-db` (see `k8s/deployment.yaml`). The app code does not fetch credentials from
Vault directly.

---

## 5. Kubernetes shape

- **Namespace:** `catalog`
- **ServiceAccount:** `catalog-service` (bound to Vault kubernetes auth role)
- **Secret delivery:** Vault Secrets Operator `VaultDynamicSecret` → K8s Secret

---

## 6. Data model

See `migrations/0001_init.sql` for the `products` table schema and seed data.

---

## 7. Local development

`docker-compose.yaml` brings up Postgres + the app. Set `DB_PASSWORD` in `.env` (see
`.env.example`). After `docker compose up`, all endpoints work against the seeded data.
