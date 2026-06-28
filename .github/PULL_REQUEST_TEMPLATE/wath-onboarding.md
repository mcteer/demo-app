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

<!-- Keep the purpose line for each file — one sentence after the em dash. -->

- [ ] `integration.params.json` — Typed source of truth — app name, Vault creds path, TTLs, K8s identity binding (schema-valid; render all other artifacts from this)
- [ ] `vault/policy.hcl` — Least-privilege Vault policy — read on `database/creds/<role>` only; no path wildcards (VDS-004)
- [ ] `vault/auth-kubernetes-role.json` — Kubernetes auth role binding — ties this app's service account and namespace to the policy (VDS-006)
- [ ] `k8s/vso-dynamic-secret.yaml` — Vault Secrets Operator CR — syncs short-lived DB credentials from Vault into a Kubernetes Secret
- [ ] `k8s/deployment.yaml` — App Deployment wired to the VSO-managed Secret — no static `DATABASE_URL` or password env values (VDS-001)
- [ ] `.github/workflows/vault-verify.yml` — CI gate re-running the standard's conformance suite on every PR (VDS-008)
- [ ] **Application code changes** — remove static credential reads; keep env var names if VSO populates them (VDS-001)
- [ ] **Removed static secrets** — no committed `Secret` manifests, DSNs, or passwords in compose/env files

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
