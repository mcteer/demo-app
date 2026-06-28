# catalog-service

A small HTTP API over a Postgres `products` table. Database credentials are delivered as
short-lived dynamic secrets from HashiCorp Vault via the Vault Secrets Operator (VSO).

## Endpoints

| Method | Path              | Description                          |
|--------|-------------------|--------------------------------------|
| GET    | `/healthz`        | Liveness probe (no database access)  |
| GET    | `/readyz`         | Readiness probe (`SELECT 1`)         |
| GET    | `/admin`          | Minimal admin UI for managing products |
| GET    | `/products`       | List all products                    |
| GET    | `/products/{id}`  | Get one product by ID                |
| POST   | `/products`       | Create a product                     |
| PUT    | `/products/{id}`  | Update a product                     |
| DELETE | `/products/{id}`  | Delete a product                     |

## Local development

Local Postgres uses trust authentication (no committed credentials). Start the stack:

```bash
docker compose up --build
```

Verify:

```bash
curl http://localhost:8080/readyz
curl http://localhost:8080/products
open http://localhost:8080/admin
```

## Configuration

All database configuration is read from environment variables in `internal/config/config.go`:

| Variable      | Example                              |
|---------------|--------------------------------------|
| `DB_HOST`     | `postgres.catalog.svc.cluster.local` |
| `DB_PORT`     | `5432`                               |
| `DB_NAME`     | `catalog`                            |
| `DB_USER`     | supplied by VSO-managed Secret       |
| `DB_PASSWORD` | supplied by VSO-managed Secret       |
| `DB_SSLMODE`  | `disable` (dev) / `require` (prod)  |
| `PORT`        | `8080` (default)                     |

In Kubernetes, `DB_USER` and `DB_PASSWORD` are sourced from the VSO-managed Secret
(`my-service-db`) — see `k8s/deployment.yaml` and `k8s/vso-dynamic-secret.yaml`.

## Kubernetes deployment

Apply integration manifests:

```bash
kubectl apply -f k8s/
```

Build and load the image locally (e.g. with kind or minikube):

```bash
docker build -t my-service:latest .
kind load docker-image my-service:latest   # if using kind
```

Check readiness:

```bash
kubectl port-forward -n my-ns-dev svc/my-service 8080:80
curl http://localhost:8080/readyz
```

## Vault integration artifacts

| File | Purpose |
|------|---------|
| `integration.params.json` | Typed source of truth for the Vault dynamic-secrets integration |
| `vault/policy.hcl` | Least-privilege Vault policy (read on `database/creds/my-service`) |
| `vault/auth-kubernetes-role.json` | Kubernetes auth role binding |
| `k8s/vso-dynamic-secret.yaml` | VSO `VaultDynamicSecret` CR |
| `k8s/deployment.yaml` | App Deployment wired to VSO-managed credentials |
| `.github/workflows/vault-verify.yml` | CI conformance gate (VDS-008) |

## Project layout

```
cmd/server/main.go              # wiring + HTTP server start
internal/config/config.go       # credential seam — reads env, builds DB config
internal/db/db.go               # pool construction, readiness check
internal/handlers/handlers.go   # HTTP handlers
internal/store/store.go         # product queries
migrations/0001_init.sql        # schema + seed
k8s/                            # Vault dynamic-secrets integration manifests
vault/                          # Vault policy and auth role definitions
deploy/                         # Legacy sandbox Postgres (dev only)
Dockerfile
docker-compose.yaml
```
