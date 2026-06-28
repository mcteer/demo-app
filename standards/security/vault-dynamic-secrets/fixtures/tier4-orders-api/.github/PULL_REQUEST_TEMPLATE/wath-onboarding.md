## Wath onboarding — Vault dynamic database secrets

<!-- Fill every section. This PR was opened by Wath to onboard this application to tier-4
     Vault dynamic database secrets. Do not delete this template file — it is reused for
     future Wath remediation PRs. -->

### Summary

<!-- 1–3 sentences: tier-1 → tier-4 migration and what changed for the app team -->

**Migration:** Replacing tier-1 static credentials with tier-4 Vault dynamic database secrets.

### Service & standard

- **Service:** HashiCorp Vault (database secrets engine)
- **Standard:** `vault-dynamic-secrets` v<!-- version -->
- **App:** <!-- app name from integration.params.json -->

### Artifacts in this PR

- [ ] `integration.params.json` (schema-valid, emitted first)
- [ ] `vault/policy.hcl` (least-privilege, rendered from params)
- [ ] `vault/auth-kubernetes-role.json` (or equivalent auth role config)
- [ ] `k8s/vso-dynamic-secret.yaml` (or Agent Injector annotations)
- [ ] `k8s/deployment.yaml` updated — **no static DSN/password in env or manifests**
- [ ] Application diff — static credential pattern removed
- [ ] `.github/workflows/vault-verify.yml` (VDS-008 durable verification)
- [ ] All static DSN / embedded password patterns removed (VDS-001)

### Verification evidence

<!-- Paste output summary or link to `.wath/verify-summary.json` / behavioral evidence -->

- [ ] Static conformance gate passed (`verify.sh` — VDS-001..008)
- [ ] `vault policy fmt` clean
- [ ] `kubeconform` clean (k8s manifests)
- [ ] Behavioral gate passed (signed identity → dynamic cred → DB connect) — if Tier-1 sandbox ran

```
<!-- paste key verify.sh output lines or attach CI run URL -->
```

### Vault admin prerequisites (human applies — Wath did NOT execute)

- [ ] `database/` secrets engine enabled for target environment(s)
- [ ] DB connection configured in Vault with **root** credential (and root rotated)
- [ ] Network path open from Vault to the database
- [ ] Database role creation statements applied / ratified by SME

### Assumptions for SME ratification

<!-- SQL grants, TTL choices, identity binding — state explicitly for human review -->

- **SQL grants assumed:**
- **Identity binding:**
- **TTL:**

### Rollout notes

<!-- Anything the owning team must do after merge: restart, pooler config, etc. -->

- [ ] No breaking changes for app team
- [ ] Requires pod restart / redeploy after merge

### Reviewer checklist

- [ ] No static credentials remain in repo (code, env, manifests, Secrets)
- [ ] Policy uses exact creds path — no wildcards (VDS-004)
- [ ] Auth role binds specific SA/namespace — no `*` (VDS-006)
- [ ] Verification evidence attached — reviewable, not vibes
