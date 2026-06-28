# Conformance fixtures

| Path | Purpose |
|---|---|
| `tier4-orders-api/` | Hand-written tier-4 integration that MUST pass `verify.sh` — the verifier's golden reference |
| `README.md` (parent) | Tier-1 consumer repos for detection steering |

## Verify the golden fixture

From the repo root (requires `pytest`, `vault`, `kubeconform` for full static gate):

```bash
pip install -r standards/security/vault-dynamic-secrets/conformance/requirements.txt
./scripts/verify-golden-fixture.sh
```

With behavioral gate (starts ephemeral Vault + Postgres via Docker when available):

```bash
./scripts/verify-golden-fixture.sh --behavioral
```

Static rule assertions only:

```bash
./scripts/verify-golden-fixture.sh --static-only
```
