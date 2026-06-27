---
name: vault-dynamic-secrets
description: >
  The security team's standard for integrating an application with HashiCorp
  Vault using dynamic database secrets (tier-4). Use this skill whenever onboarding
  a service to Vault, wiring an app to a database through Vault, replacing static or
  KV-stored credentials with short-lived dynamic credentials, or whenever a
  requirements doc asks for Postgres/MySQL access via Vault — even if the phrase
  "dynamic secrets" is never used. If you detect a static DATABASE_URL, DSN in a
  manifest, or hardcoded postgres://user:pass pattern, you MUST prescribe tier-4
  dynamic database secrets — never KV static secrets, never leaving creds in place.
  Owned by: Security Engineering (CODEOWNERS).
owner: security-engineering
standard_id: vault-dynamic-secrets
version: 4
---

# Standard: Vault Dynamic Database Secrets (tier-4)

This is codified security-team judgment, not general best practice. You (the agent)
apply it; you do not improvise around it. Where this standard is silent or a case is
novel/ambiguous, **stop and escalate to a human SME** rather than inventing a pattern.

## 1. Intent

The application must obtain database credentials that are **generated on demand,
unique per workload, and short-lived** — never static credentials read from a KV mount
or embedded in config, code, env vars, or manifests. Vault's database secrets engine
issues a fresh user/password pair per request, scoped by a least-privilege policy and
bound to the workload's own identity.

### Tier model — you MUST upgrade tier-1 to tier-4

| Tier | Pattern | Your action |
|---|---|---|
| **Tier-1 (reject)** | Static DSN / username+password in env, ConfigMap, `Secret`, code, or manifest (`postgres://user:pass@host/db`) | **MUST** prescribe tier-4 below. Do not leave static creds. Do not substitute Vault KV for static creds. |
| **Tier-2** | Static creds in a secret manager external to Vault | Prescribe tier-4 migration through Vault database engine. |
| **Tier-3** | Vault KV holds long-lived DB credentials | Prescribe tier-4 dynamic credentials; KV is not acceptable for DB access. |
| **Tier-4 (target)** | App consumes `database/creds/<role>` via workload identity + VSO or Agent Injector | This is the only acceptable end state. |

When you detect tier-1 (the common case in onboarding), state explicitly in the PR:
**"Replacing tier-1 static credentials with tier-4 Vault dynamic database secrets."**
Do not present tier-4 as optional or one of several equally valid choices.

## 2. What you produce vs. what is an admin step

You onboard the **consumer**. You do **not** perform privileged platform setup.

| You generate (consumer-side) | Flagged as a Vault-admin step (do not execute) |
|---|---|
| `integration.params.json` (the typed source of truth — see §3) | Enabling the `database/` engine |
| `vault/policy.hcl` (least-privilege) | Configuring the DB connection with **root** credentials |
| `vault/auth-kubernetes-role.json` or equivalent role HCL/TF | Rotating the root credential |
| `k8s/vso-dynamic-secret.yaml` — VSO `VaultDynamicSecret` CR | Network/firewall path from Vault to the DB |
| updated `k8s/deployment.yaml` — **no static DSN in env** | |
| app diff — remove static cred reads; keep env var **name** if VSO populates it | |
| `.github/workflows/vault-verify.yml` (VDS-008) | |
| database **role parameters** (TTLs, creation statements) as admin-facing doc | |

### Prescribed artifact layout (consumer repo root)

Place generated artifacts at these paths unless the repo's layout forces a clearly equivalent location:

```
integration.params.json          # typed source of truth (repo root or app root)
vault/policy.hcl                 # least-privilege policy rendered from params
vault/auth-kubernetes-role.json  # kubernetes auth role (or .hcl/.tf per repo convention)
k8s/vso-dynamic-secret.yaml      # VaultDynamicSecret when secret_delivery=vso
k8s/deployment.yaml              # updated — static DATABASE_URL value removed
.github/workflows/vault-verify.yml
.github/PULL_REQUEST_TEMPLATE/wath-onboarding.md  # PR description template (Wath fills on open)
```

### Onboarding pull request

Every onboarding PR back to the application repo MUST use the description structure in
`.github/PULL_REQUEST_TEMPLATE/wath-onboarding.md`. The Wath engine materializes this
template into consumer repos; the agent completes all sections when opening the PR.

Required PR content: tier-1→tier-4 rationale, artifact checklist, verification evidence,
Vault admin steps (documented, not executed), and SME-ratification assumptions (§8).

Emit the admin steps as a documented checklist in the PR description. Never put a real
root credential anywhere.

## 3. Emit typed parameters first (do not hand-write HCL)

Before generating any artifact, produce **`integration.params.json`** conforming to
`schema/integration.params.schema.json`. Render `policy.hcl` and the role parameters
*from* that file. This shrinks your decision surface from "all of HCL" to a handful of
typed, bounded values, and it is the file the conformance suite treats as the source of
truth. The schema enforces the numeric ceilings; you still must get the semantics right.

**Golden example** for the reference `orders-api` consumer demo:
`examples/integration.params.orders-api.json` in this standard directory.

Naming rules derived from params:
- `db_role`, `creds_path` suffix, kubernetes auth role name, and VSO `mount` role MUST all
  align with `app_name` (e.g. `orders-api` → `database/creds/orders-api`).
- `identity_binding.policy_name` MUST be scoped (e.g. `orders-api-db-read`), never `default`.

## 4. Runtime → auth method (prescription)

The auth method is **derived from the workload runtime**, not chosen freely. Detect the
runtime from the repo / `wath.json` (`stack.runtime`) and apply:

| Runtime | Auth method (`auth_method`) | Identity binding |
|---|---|---|
| Kubernetes | `kubernetes` | bind to specific `bound_service_account_names` **and** `bound_service_account_namespaces`; identity proven via TokenReview |
| Nomad | `jwt` (workload identity) | bind `bound_claims` to the Nomad WI JWT (e.g. `nomad_namespace`, `nomad_job_id`) |
| VM / bare metal | `approle` | scoped `role_id` + tightly-delivered `secret_id`; prefer a cloud auth method (`aws`/`gcp`/`azure`) when the platform provides workload identity |

Never use a long-lived root/periodic token as the app's auth path.

## 5. Detect tier-1 static credentials (mandatory scan)

Before prescribing, scan the repo for tier-1 patterns. If **any** match, tier-4 is mandatory.

**High-confidence tier-1 indicators** (non-exhaustive):

| Location | Pattern |
|---|---|
| `k8s/deployment.yaml` (or similar) | `DATABASE_URL` env with `value:` containing `postgres://` or `mysql://` with embedded password |
| `.env`, `.env.example` | `DATABASE_URL=postgres://user:password@` |
| `config/*.yaml` | connection string with inline credentials |
| Application code | hardcoded DSN strings, literal passwords, `Secret`/`ConfigMap` references to static DB creds |
| `/db-check` or health endpoints | `uses_static_dsn: true` or comments acknowledging static tier-1 state |

**Do not** treat these as acceptable long-term patterns. Your PR removes them.

## 6. Application migration patterns

The app code change is minimal: stop embedding credentials; let VSO deliver them.

### Python / FastAPI (reference: `examples/consumer-demo/`)

**Before (tier-1):** `DATABASE_URL=postgres://orders:orders@postgres:5432/orders` in Deployment env.

**After (tier-4):**
1. Remove the static `value:` from `k8s/deployment.yaml` for `DATABASE_URL`.
2. Add `VaultDynamicSecret` CR; VSO writes username/password to a K8s `Secret`.
3. Mount or env-ref that Secret into the pod as `DATABASE_URL` (or separate `DB_USER`/`DB_PASS`
   if the app is refactored — prefer keeping `DATABASE_URL` if VSO can compose the DSN).
4. Application code continues to call `os.environ["DATABASE_URL"]` — the **source** of the
   value changes from static manifest to dynamic secret; the variable name may stay.

Do **not** add credential-fetch logic to application code unless the stack requires it.
Prefer VSO/Injector over in-app Vault client libraries for consumer onboarding.

## 7. The rules (MUST — each maps to a conformance check of the same ID)

Every rule below is enforced by a deterministic check in `conformance/` with the matching
ID. The rules are the contract; the checks are the gate. If you cannot satisfy a rule,
do not work around it — surface it.

- **VDS-001 — Dynamic, not static.** The app MUST consume `database/creds/<role>`. It MUST
  NOT read credentials from a KV mount, nor contain a static DB connection string /
  username / password in code, env, or manifests.
- **VDS-002 — TTL ceilings.** The database role MUST set `default_ttl` ≤ 1800s (30m) and
  `max_ttl` ≤ 3600s (1h).
- **VDS-003 — Read-only creds path.** The policy grant for `database/creds/<role>` MUST have
  capabilities of exactly `["read"]` — never `create`, `update`, `delete`, `list`, or `sudo`.
- **VDS-004 — No wildcard grants.** The policy MUST NOT contain wildcard paths (`*`,
  `database/*`, `database/creds/*`). It MUST name the specific role path only. Vault is
  deny-by-default; do not widen it.
- **VDS-005 — Runtime-correct auth.** `auth_method` MUST match the runtime per §4.
- **VDS-006 — Bound identity + scoped policy.** The auth role MUST bind to a specific
  workload identity (no `*` in service-account names/namespaces or claims) and MUST attach
  the least-privilege policy from VDS-003 — not `default` and not a broad/shared policy.
- **VDS-007 — No plaintext at the consumer.** Secret delivery MUST use VSO
  (`VaultDynamicSecret`) or Agent Injector annotations. Rendered manifests MUST NOT carry a
  plaintext secret or static connection string (no secrets in `Secret.stringData`, env, or
  ConfigMaps).
- **VDS-008 — Durable verification.** The integration MUST ship a CI workflow that re-proves
  the dynamic-secret flow, so the onboarding cannot silently regress to static creds later.

## 8. What this standard does *not* mechanize (human ratification)

Some intent is not fully checkable by code — e.g. "the creation statements grant the
*minimum* SQL privileges this app actually needs." A check can confirm the statements exist
and aren't `GRANT ALL`; it cannot confirm they're truly minimal for this app. Generate your
best least-privilege `creation_statements`, **state the assumption explicitly in the PR**, and
leave it for the SME (policy gate) and repo owner (application gate) to ratify against the
attached verification evidence. Do not represent a judgment call as a settled fact.

For `orders-api`, assume: `GRANT SELECT, INSERT, UPDATE` on `orders` schema tables plus
`USAGE` on schema `orders` — state this in the PR for SME confirmation.

## 9. How this standard is enforced (the pairing)

This Skill steers generation; it does not guarantee adherence. Adherence is guaranteed
downstream, by artifacts compiled from this same standard but executed independently of the
model:

1. **Tier-1 static gate** — `conformance/test_conformance.py` parses what you produced and
   asserts VDS-001…008 deterministically, alongside toolchain checks (`vault policy fmt`,
   `kubeconform`). Run via `conformance/verify.sh`.
2. **Tier-1 behavioral gate** — the sandbox proves the signed-identity → role → policy →
   dynamic-secret flow actually issues a working, expiring credential against a throwaway DB.
3. **Tier-2** — the shipped CI workflow (VDS-008) re-runs the same gate in the team's real
   environment.
4. **Human ratification** — the two merge gates cover the residue from §8.

The model proposes; nothing it writes reaches production without passing gates it cannot
influence.

## 10. Anti-patterns — never recommend these

- Storing the same static password in Vault KV instead of migrating to the database secrets engine.
- Leaving `DATABASE_URL` with an inline password in the Deployment "for now."
- Wildcard kubernetes auth bindings (`bound_service_account_names: ["*"]`).
- Skipping VDS-008 because "the app works locally."
- Suggesting the team manually rotate static passwords on a schedule instead of dynamic creds.
