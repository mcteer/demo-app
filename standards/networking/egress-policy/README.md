# Prepped extension standard (not in active registry)

This standard is **ready to enable** for cold-extension demos. See [docs/extensions/README.md](../../../../docs/extensions/README.md).

To activate, add an entry to `standards/registry.yaml` and re-run onboarding with `--standard-id egress-policy`.

Minimal params example:

```json
{
  "application": "orders-api",
  "runtime": "kubernetes",
  "egress": [
    { "host": "vault.example.com", "port": 8200, "protocol": "tcp", "purpose": "Vault API" },
    { "host": "postgres.example.com", "port": 5432, "protocol": "tcp", "purpose": "Database" }
  ]
}
```
