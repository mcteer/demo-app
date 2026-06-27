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
    db-credentials.secret.yaml    # THE STATIC SECRET — tier-1 "before" (see §5)
    deployment.yaml               # app Deployment, runs as the catalog-service SA
    service.yaml                  # ClusterIP Service
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
DB_USER       e.g. catalog_app
DB_PASSWORD   e.g. (static password, tier-1)
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

## 5. The deliberate tier-1 state (build this, do not "fix" it)

Credentials are delivered as a **static Kubernetes Secret** that the Deployment mounts as
environment variables. This is the pattern Wath will later replace. Build it exactly:

```yaml
# deploy/db-credentials.secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
  namespace: catalog
type: Opaque
stringData:
  DB_USER: catalog_app
  DB_PASSWORD: (supplied at runtime via VSO-managed Secret — not committed)
```

The Deployment sources these via `envFrom: [{ secretRef: { name: db-credentials } }]` (plus the
non-secret `DB_HOST`/`DB_NAME`/etc. as plain env). The password is static, long-lived, and
committed as a manifest — exactly the realistic "before" the demo improves on. **Leave it that
way.**

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

## 9. Signals the onboarding will detect (so the demo reads well)

You don't act on these, but building them faithfully makes the live detection step legible. The
onboarding keys on: a **static credential** in `db-credentials.secret.yaml`, the app reading
`DB_USER`/`DB_PASSWORD` from env, **no Vault wiring of any kind**, a read-only data access pattern,
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
5. There is **zero** reference to Vault, dynamic secrets, VSO, the Agent Injector, AppRole, or
   Kubernetes auth anywhere in the repo.

---

## 11. Non-goals — do **not** build these (this is the demo)

- **No HashiCorp Vault integration.** No Vault client, no `vault` config, no policies, no roles.
- **No dynamic secrets, no Vault Secrets Operator, no Agent Injector annotations.**
- **No secret-management refactor.** The static Secret stays static.
- **No auth method setup** (Kubernetes auth, AppRole, JWT/workload identity).
- **No CI workflow for secret verification.**
- **No `.cursor/` onboarding artifacts** (rules, `mcp.json`, requirements). Those belong to the
  Wath onboarding layer and are added separately.

If you find yourself improving how this app handles credentials, stop — that improvement *is* the
demo, and it is performed live by Wath, not by you.

---

## 12. How this fits the larger demo (FYI, not build scope)

This repository is the **consumer repo**. After it exists in this tier-1 state, the Wath onboarding
layer is added on top: an `INTEGRATION_REQUIREMENTS.md` describing this app's environment and
intent, the `.cursor/rules/*.mdc` and `.cursor/mcp.json` that let `@wath onboard` run here, and —
produced live during the demo — the integration PR (Vault policy, auth role, `VaultDynamicSecret`
CR, updated Deployment wiring, and a shipped verification workflow) that carries this service from
the static Secret above to short-lived, dynamic, least-privilege database credentials. None of that
is your concern for this build. Deliver the clean tier-1 service.
