# catalog-service

A small HTTP API over a Postgres `products` table. Database credentials are delivered via
HashiCorp Vault dynamic secrets through the Vault Secrets Operator (tier-4).

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
| `DB_USER`     | `catalog_app`                        |
| `DB_PASSWORD` | supplied by VSO-managed Secret       |
| `DB_SSLMODE`  | `disable` (demo) / `require` (prod)  |
| `PORT`        | `8080` (default)                     |

## Kubernetes deployment

Apply manifests (includes dev Postgres for sandbox/demo):

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
deploy/                         # Kubernetes manifests
Dockerfile
docker-compose.yaml
```
