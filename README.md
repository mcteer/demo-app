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

Copy the example env file and start the stack:

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
| `DB_USER`     | from VSO-managed Secret (local: see `.env.example`) |
| `DB_PASSWORD` | from VSO-managed Secret (local: see `.env.example`) |
| `DB_SSLMODE`  | `disable` (demo) / `require` (prod)  |
| `PORT`        | `8080` (default)                     |

## Kubernetes deployment

Apply the Vault dynamic-secrets integration manifests:

```bash
kubectl apply -f k8s/
```

For the legacy sandbox Postgres in namespace `catalog`:

```bash
kubectl apply -f deploy/
```

Build and load the image locally (e.g. with kind or minikube):

```bash
docker build -t catalog-service:latest .
kind load docker-image catalog-service:latest   # if using kind
```

Apply the migration manually if not using the bundled Postgres init ConfigMap:

```bash
kubectl exec -n catalog deploy/postgres -- psql -U catalog_app -d catalog -f /docker-entrypoint-initdb.d/0001_init.sql
```

Check readiness:

```bash
kubectl port-forward -n catalog svc/catalog-service 8080:80
curl http://localhost:8080/readyz
```

## Project layout

```
cmd/server/main.go              # wiring + HTTP server start
internal/config/config.go       # credential seam — reads env, builds DB config
internal/db/db.go               # pool construction, readiness check
internal/handlers/handlers.go   # HTTP handlers
internal/store/store.go         # product queries
migrations/0001_init.sql        # schema + seed
deploy/                         # Legacy sandbox Kubernetes manifests
k8s/                            # Vault dynamic-secrets integration manifests
vault/                          # Vault policy and auth role (admin-facing)
integration.params.json         # Typed integration source of truth
Dockerfile
docker-compose.yaml
```
