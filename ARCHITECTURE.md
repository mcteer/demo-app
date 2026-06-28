# catalog-service — application architecture & build spec

> **For the building agent:** this document specifies a small, realistic Postgres-backed
> microservice. The service consumes short-lived database credentials from HashiCorp Vault via
> the Vault Secrets Operator (tier-4 dynamic secrets).

---

## 0. Why this app exists (context, not scope)

This service is the **target of a live onboarding demo**. A separate system ("Wath") onboarded
this app from static database credentials to HashiCorp Vault **dynamic** secrets. The integration
artifacts (`integration.params.json`, `vault/`, `k8s/vso-dynamic-secret.yaml`) live in this repo.

---

## 1. What the app is

`catalog-service` is a small HTTP API over a Postgres `products` table. It is intentionally
mundane — a read-mostly catalog endpoint of the kind every company has a dozen of. Realism is the
point; do not make it clever.

**Behaviour:**
- `GET /healthz` — liveness. Returns `200 {"status":"ok"}`. Must **not** touch the database.
- `GET /readyz` — readiness. Executes `SELECT 1` against Postgres; `200` if it succeeds, `503`
  otherwise. This is the signal the demo's verification uses to prove the app can connect with
  whatever credentials it currently holds.
- `GET /products` — returns all rows from `products` as JSON.
- `GET /products/{id}` — returns one row, or `404`.

The app only ever **reads**. It issues no writes. (This justifies read-only database credentials
downstream and matches the integration's least-privilege posture.)

---

## 2. Stack

- **Language:** Go (1.22+). Chosen for a single static binary, a tiny container, and credibility
  with an infrastructure-literate audience. *If you prefer Python (FastAPI) or Node (Express),
  the rest of this spec is stack-neutral — keep every interface, env var name, path, and the
  credential seam identical.*
- **Postgres driver:** `pgx` (v5) via `database/sql`, or `jackc/pgx` pool directly.
- **HTTP:** standard library `net/http` with a lightweight router (`chi` is fine). No heavy
  framework.
- **Config:** environment variables only (see §4). No config files for secrets.
- **Container:** multi-stage `Dockerfile`, distroless or `alpine` final image.

---

## 3. Repository layout

```
catalog-service/
  cmd/server/main.go              # wiring + HTTP server start
  internal/config/config.go       # THE CREDENTIAL SEAM — reads env, builds DB config
  internal/db/db.go               # pool construction, readiness check
  internal/handlers/handlers.go   # the four HTTP handlers
  internal/store/store.go         # product queries
  migrations/0001_init.sql        # schema + seed
  k8s/
    namespace.yaml                # namespace: catalog
    serviceaccount.yaml           # ServiceAccount: catalog-service
    vso-dynamic-secret.yaml       # VaultDynamicSecret CR (VSO)
    deployment.yaml               # app Deployment — creds from VSO-managed Secret
    service.yaml                  # ClusterIP Service
  deploy/
    postgres.yaml                 # dev Postgres Deployment + Service (demo/sandbox only)
  Dockerfile
  docker-compose.yaml             # local: app + postgres, static creds
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
DB_USER       from VSO-managed Secret (key: username)
DB_PASSWORD   from VSO-managed Secret (key: password)
DB_SSLMODE    e.g. disable (demo) / require (prod)
PORT          HTTP listen port, default 8080
```

**Design requirement — keep credential sourcing isolated.** `config.go` is the *only* file that
reads `DB_USER`/`DB_PASSWORD`. `db.go` receives a struct, never the environment. This is
deliberate: the live onboarding changes *how those two values are supplied* (a rotating, Vault-fed
Secret instead of a static one), and a contained seam keeps that diff small and legible. Do not
scatter `os.Getenv("DB_PASSWORD")` across the codebase.

**Connection handling:** build the pool from the config at startup. Treat credentials as
re-readable rather than eternal — if the pool errors with an auth failure, it is acceptable to
rebuild it from the current environment. (Full live-rotation handling is a stretch goal, not a
requirement; the demo injects credentials before the pod starts.)

---

## 5. Tier-4 credential delivery (Vault + VSO)

Credentials are **not** committed to the repo. The Vault Secrets Operator syncs short-lived
`database/creds/catalog-service` credentials into the `catalog-service-db` Kubernetes Secret.
The Deployment references `username` and `password` keys via `secretKeyRef`.

---

## 6. Kubernetes shape that makes onboarding clean

Two details in the "before" manifests exist so the later onboarding is tidy. Include them now:

- **Dedicated ServiceAccount.** The Deployment runs as a named ServiceAccount `catalog-service`
  (not `default`). Kubernetes auth binds a Vault role to a specific SA + namespace, so the app
  having its own identity from day one is both realistic and what makes the onboarding bind to
  something real.
- **Named namespace.** Everything lives in namespace `catalog`. No use of `default`.

Everything else is ordinary: a `Deployment` (1 replica is fine for the demo), a `ClusterIP`
`Service` on port 80→8080, resource requests/limits, and the `/healthz`–`/readyz` probes wired as
liveness/readiness.

---

## 7. Data model

```sql
-- migrations/0001_init.sql
CREATE TABLE IF NOT EXISTS products (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    sku         TEXT NOT NULL UNIQUE,
    price_cents INTEGER NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO products (name, sku, price_cents) VALUES
    ('Field Notebook',  'FN-001', 1200),
    ('Ford Crossing Mug','FC-002', 1800),
    ('Ferryman Tee',    'FT-003', 2500)
ON CONFLICT (sku) DO NOTHING;
```

Apply the migration via an init step / `docker-entrypoint-initdb.d` mount for local, and document a
one-liner for the cluster. The app does not need a migration framework.

---

## 8. Local development

`docker-compose.yaml` brings up Postgres + the app with the **same static-credential pattern** as
the cluster (env from a compose `environment:` block standing in for the Secret). After
`docker compose up`, all four endpoints work and `/products` returns the three seed rows. This is
how you self-check the build.

---

## 9. Integration signals

The onboarding layer keys on: **Vault dynamic secrets** via VSO, the app reading
`DB_USER`/`DB_PASSWORD` from a VSO-managed Secret, a read-only data access pattern (with CRUD
handlers for demo), a dedicated ServiceAccount, and a declared runtime of Kubernetes.

---

## 10. Acceptance criteria

1. `docker compose up` yields a service whose `/readyz` returns `200` and `/products` returns the
   seeded rows.
2. The image builds via the `Dockerfile`; the binary runs with only the §4 env vars set.
3. `kubectl apply -f deploy/` (with a cluster + the dev Postgres) yields a running pod under the
   `catalog-service` ServiceAccount in namespace `catalog`, passing readiness.
4. Credential reading is confined to `internal/config/config.go`.
5. Vault integration artifacts are present under `vault/`, `k8s/`, and `integration.params.json`.

---

## 11. Non-goals

- **No in-app Vault client.** Credential fetch is delegated to VSO; the app reads env vars only.
- **No static credentials in repo.** No committed Secrets, DSNs, or long-lived passwords.

---

## 12. How this fits the larger demo (FYI, not build scope)

This repository is the **consumer repo**. After it exists in this tier-1 state, the Wath onboarding
layer is added on top: an `INTEGRATION_REQUIREMENTS.md` describing this app's environment and
intent, the `.cursor/rules/*.mdc` and `.cursor/mcp.json` that let `@wath onboard` run here, and —
produced live during the demo — the integration PR (Vault policy, auth role, `VaultDynamicSecret`
CR, updated Deployment wiring, and a shipped verification workflow) that carries this service from
the static Secret above to short-lived, dynamic, least-privilege database credentials. None of that
is your concern for this build. Deliver the clean tier-1 service.
