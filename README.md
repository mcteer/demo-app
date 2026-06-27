# catalog-service

A small HTTP API over a Postgres `products` table. Database credentials are delivered via
HashiCorp Vault dynamic secrets (tier-4) through the Vault Secrets Operator.

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

Copy `.env.example` to `.env` and set `DB_PASSWORD` for local Postgres:

```bash
cp .env.example .env
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
| `DB_USER`     | (from VSO-managed Secret)            |
| `DB_PASSWORD` | (from VSO-managed Secret)            |
| `DB_SSLMODE`  | `disable` (demo) / `require` (prod)  |
| `PORT`        | `8080` (default)                     |

## Kubernetes deployment

Apply manifests (requires Vault Secrets Operator and platform admin prerequisites):

```bash
kubectl apply -f k8s/
```

Build and load the image locally (e.g. with kind or minikube):

```bash
docker build -t catalog-service:latest .
kind load docker-image catalog-service:latest   # if using kind
```

Check readiness:

```bash
kubectl port-forward -n catalog svc/catalog-service 8080:80
curl http://localhost:8080/readyz
```

## Vault integration artifacts

| Path | Purpose |
|------|---------|
| `integration.params.json` | Typed source of truth for the integration |
| `vault/policy.hcl` | Least-privilege Vault policy |
| `vault/auth-kubernetes-role.json` | Kubernetes auth role binding |
| `k8s/vso-dynamic-secret.yaml` | VSO `VaultDynamicSecret` CR |
| `.github/workflows/vault-verify.yml` | Durable conformance verification (VDS-008) |

## Project layout

```
cmd/server/main.go              # wiring + HTTP server start
internal/config/config.go       # credential seam — reads env, builds DB config
internal/db/db.go               # pool construction, readiness check
internal/handlers/handlers.go   # HTTP handlers
internal/store/store.go         # product queries
migrations/0001_init.sql        # schema + seed
k8s/                            # Kubernetes manifests (tier-4)
vault/                          # Vault policy and auth role config
integration.params.json         # Wath integration parameters
Dockerfile
docker-compose.yaml
```
