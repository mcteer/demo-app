# catalog-service — application architecture

> **Tier-4:** This service consumes short-lived database credentials from HashiCorp Vault
> via the Vault Secrets Operator. See `integration.params.json` and `k8s/` for the
> vault-dynamic-secrets integration.

---

## 1. What the app is

`catalog-service` (integrated as `my-service` in Wath) is a small HTTP API over a Postgres
`products` table.

**Behaviour:**
- `GET /healthz` — liveness. Returns `200 {"status":"ok"}`. Must **not** touch the database.
- `GET /readyz` — readiness. Executes `SELECT 1` against Postgres; `200` if it succeeds, `503`
  otherwise.
- `GET /products` — returns all rows from `products` as JSON.
- `GET /products/{id}` — returns one row, or `404`.
- CRUD endpoints for product management via `/admin` UI.

---

## 2. Stack

- **Language:** Go 1.22+
- **Postgres driver:** `jackc/pgx` v5 pool
- **HTTP:** `net/http` with `chi` router
- **Config:** environment variables only (see below)
- **Secrets:** Vault dynamic database credentials via VSO

---

## 3. Configuration contract (the credential seam)

All database configuration is read **in one place** — `internal/config/config.go` — from these
environment variables:

```
DB_HOST       e.g. postgres.my-ns-dev.svc.cluster.local
DB_PORT       e.g. 5432
DB_NAME       e.g. catalog
DB_USER       supplied by VSO-managed Secret at runtime
DB_PASSWORD   supplied by VSO-managed Secret at runtime
DB_SSLMODE    e.g. disable (dev) / require (prod)
PORT          HTTP listen port, default 8080
```

**Design requirement — keep credential sourcing isolated.** `config.go` is the *only* file that
reads `DB_USER`/`DB_PASSWORD`. `db.go` receives a struct, never the environment.

In Kubernetes, credentials are delivered by VSO into a Secret (`my-service-db`); the Deployment
references that Secret via `secretKeyRef` — no static credentials in manifests.

---

## 4. Repository layout

```
cmd/server/main.go
internal/config/config.go       # THE CREDENTIAL SEAM
internal/db/db.go
internal/handlers/handlers.go
internal/store/store.go
migrations/0001_init.sql
integration.params.json         # Vault integration source of truth
vault/                          # Vault policy + auth role
k8s/                            # VSO CR + Deployment
deploy/                         # Sandbox Postgres (dev only)
Dockerfile
docker-compose.yaml
```

---

## 5. Local development

`docker-compose.yaml` brings up Postgres with trust authentication (no committed credentials).
The app connects with `DB_USER=catalog_app` and an empty `DB_PASSWORD`.

```bash
docker compose up --build
```

---

## 6. Kubernetes identity

The integration binds Vault Kubernetes auth to:
- **Service account:** `my-service`
- **Namespaces:** `my-ns-dev`, `my-ns-prod`

See `vault/auth-kubernetes-role.json` and `k8s/serviceaccount.yaml`.
