---
name: egress-policy
description: >
  Load when onboarding Kubernetes or Nomad workloads that require explicit egress allowlists.
  Prescribes NetworkPolicy / Nomad service-mesh egress rules derived from integration.params.json.
owner: network-engineering
standard_id: egress-policy
version: 1
---

# Standard: egress-policy

Prescriptive egress allowlisting for workloads onboarded via Wath. Complements secret-delivery
standards (e.g. vault-dynamic-secrets) — it does not replace them.

## 1. Intent

Every onboarded workload gets an explicit **egress allowlist** matching its declared dependencies
(Vault, database, observability). Default deny; no `0.0.0.0/0` except where SME-ratified.

## 2. Artifact layout

| Artifact | Purpose |
|----------|---------|
| `integration.params.json` | Typed egress destinations + ports |
| `k8s/networkpolicy-egress.yaml` | Kubernetes NetworkPolicy (when runtime=kubernetes) |
| `nomad/egress.hcl` | Nomad service-mesh egress stanza (when runtime=nomad) |

## 3. Emit typed parameters first

Produce **`integration.params.json`** conforming to `schema/integration.params.schema.json` before
rendering manifests.

## 4. The rules (MUST)

- **EGR-001** — Every declared dependency has an explicit egress rule; no wildcard `*` destinations.
- **EGR-002** — Default deny: policy must not allow broad internet egress unless documented in PR for SME ratification.

## 5. How this standard is enforced

1. **Static gate** — `conformance/test_conformance.py` + `conformance/verify.sh`
2. **Tier-2** — optional CI workflow re-running the gate
3. **Human ratification** — NetEng SME + repo owner merge
