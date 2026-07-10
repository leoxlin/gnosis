---
type: Repository Delta
title: Skill names and purpose refinement
description: Shortened skill invocation names, added single-concept ingest, and made repository purpose one high-level file with sub-purposes.
tags: [delta, plugins, skills, ingest, purpose]
timestamp: 2026-07-10T12:09:54Z
status: completed
---

# Fulfilled directives

- User request to refine `gnosis-vault` and `gnosis-bootstrap` skills and constrain repository purpose.

# Change summary

- Renamed multi-concept ingest to `ingest-knowledge` and added `ingest-concept` for exactly one concept-page target.
- Renamed the bootstrap reasoning skill to `reason-with-knowledge` and shortened the four recorders to `record-decision`, `record-delta`, `record-directive`, and `record-purpose`.
- Made repository purpose a single concise, high-level file with sub-purposes and aligned its concept definition and scaffold template.

# Verification

- Every changed or added skill passes `quick_validate.py`.
- `mise run checks`

# Deviations

None.

# Related decisions

- [Bootstrap `gnosis` knowledge first on OKF](../decisions/bootstrap-knowledge-first.md)
