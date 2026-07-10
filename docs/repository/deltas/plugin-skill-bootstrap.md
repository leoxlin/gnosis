---
type: Repository Delta
title: Plugin skill bootstrap
description: Added focused vault-management and knowledge-first repository-recording skills to the gnosis plugins.
tags: [delta, plugins, skills, vault, bootstrap]
timestamp: 2026-07-10T01:03:05Z
status: completed
---

# Fulfilled directives

- User request to bootstrap focused skills for `gnosis-vault` and `gnosis-bootstrap`.

# Change summary

- Added vault skills for ingest, read-only query, and structural or semantic maintenance.
- Added concise recorders for repository purpose, decisions, directives, and deltas; each defers ontology detail to `docs/concepts/`.
- Trimmed the core bootstrap skill to route through recorded purpose, decisions, concepts, and the specialized recorders.

# Verification

- Every changed or added skill passes `quick_validate.py`.
- `mise run check`

# Deviations

Setup, capture, cross-linking, deduplication, and status were folded into CLI or core maintenance workflows instead of becoming separate skills.

# Related decisions

- [Bootstrap `gnosis` knowledge first on OKF](../decisions/bootstrap-knowledge-first.md)
