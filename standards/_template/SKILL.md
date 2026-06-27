---
name: {{STANDARD_ID}}
description: >
  TODO: Describe when an agent should load this standard.
owner: {{OWNER}}
standard_id: {{STANDARD_ID}}
version: 1
---

# Standard: {{STANDARD_ID}}

TODO: Author SME rules here. Each rule MUST have a matching conformance test
named `test_{{RULE_PREFIX}}_NNN_...` in conformance/test_conformance.py.

## 1. Intent

TODO: What this integration achieves in plain language.

## 2. What you produce vs. what is an admin step

TODO: Consumer vs. admin boundary table.

## 3. Emit typed parameters first

Before generating any artifact, produce **`integration.params.json`** conforming to
`schema/integration.params.schema.json`.

## 4. The rules (MUST)

- **{{RULE_PREFIX}}-001** — TODO
- **{{RULE_PREFIX}}-002** — TODO

## 5. How this standard is enforced

1. **Static gate** — `conformance/test_conformance.py` + `conformance/verify.sh`
2. **Tier-2** — shipped CI workflow in the consumer repo
3. **Human ratification** — SME + repo owner merge gates
