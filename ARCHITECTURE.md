# catalog-service — application architecture & build spec

> **For the building agent:** this document specifies a small, realistic Postgres-backed
> microservice. Build it exactly to the state described here — **tier-1 (static credentials)** —
> and **stop**. Read the *Non-goals* section before you start; the most important instruction in
> this file is what **not** to build.

---

## 0. Why this app exists (context, not scope)

This service is the **target of a live onboarding demo**. A separate system ("Wath") will, on
stage, onboard this app from static database credentials to HashiCorp Vault **dynamic** secrets.
For that demo to work, this app must *begin* in the state a real team would actually be in before
they adopt dynamic secrets: a normal service that reads a Postgres database using a **static
connection string delivered through a Kubernetes Secret**.

Your job is to build that believable "before." You are **not** integrating Vault. The value of the
demo is watching Wath add that later; if it is already here, there is nothing to demonstrate.

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
  deploy/
    namespace.yaml                # namespace: catalog
    serviceaccount.yaml           # ServiceAccount: catalog-service  (see §6 — keep this!)
    postgres.yaml                 # dev Postgres Deployment + Service (demo/sandbox only)
    deployment.yaml               # app Deployment — DB creds via VSO-managed Secret
    service.yaml                  # ClusterIP Service
  Dockerfile
  docker-compose.yaml             # local: app + postgres, env-supplied creds
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
DB_USER       e.g. catalog_app
DB_PASSWORD   supplied at runtime by VSO-managed Secret
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

## 5. Credential delivery (tier-4)

Credentials are delivered by the Vault Secrets Operator (`VaultDynamicSecret`) into a Kubernetes
Secret (`my-service-db`). The Deployment sources `DB_USER` and `DB_PASSWORD` via
`secretKeyRef` (plus non-secret `DB_HOST`/`DB_NAME`/etc. as plain env). No static credentials
are committed to the repository.

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

`docker-compose.yaml` brings up Postgres + the app with credentials supplied via environment
variables (standing in for the VSO-managed Secret in cluster). After
`docker compose up`, all four endpoints work and `/products` returns the three seed rows. This is
how you self-check the build.

---

## 9. Signals the onboarding will detect (so the demo reads well)

You don't act on these, but building them faithfully makes the live detection step legible. The
onboarding keys on: credentials sourced from a VSO-managed Secret, the app reading
`DB_USER`/`DB_PASSWORD` from env, Vault dynamic-secret wiring in `k8s/`, and a read-write data access pattern,
a dedicated ServiceAccount, and a declared runtime of Kubernetes. Keep all of these true and
unambiguous.

---

## 10. Acceptance criteria

1. `docker compose up` yields a service whose `/readyz` returns `200` and `/products` returns the
   seeded rows.
2. The image builds via the `Dockerfile`; the binary runs with only the §4 env vars set.
3. `kubectl apply -f deploy/` (with a cluster + the dev Postgres) yields a running pod under the
   `catalog-service` ServiceAccount in namespace `catalog`, passing readiness.
4. Credential reading is confined to `internal/config/config.go`.
5. Vault integration artifacts (`integration.params.json`, `vault/`, `k8s/`, `.github/workflows/vault-verify.yml`)
   are present and conform to the vault-dynamic-secrets standard.

---

## 11. Vault integration artifacts

This repository ships tier-4 Vault dynamic database secret integration:

- `integration.params.json` — typed source of truth
- `vault/policy.hcl` — least-privilege read on `database/creds/my-service`
- `vault/auth-kubernetes-role.json` — Kubernetes auth role bound to `my-service` SA
- `k8s/vso-dynamic-secret.yaml` — VSO `VaultDynamicSecret` CRs for dev and prod
- `k8s/deployment.yaml` — Deployment wired to VSO-managed Secret
- `.github/workflows/vault-verify.yml` — durable conformance verification (VDS-008)
