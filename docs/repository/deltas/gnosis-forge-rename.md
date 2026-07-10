---
type: Repository Delta
title: "`gnosis-forge` rename"
description: Renamed the knowledge-driven software-development bundle from `gnosis-bootstrap` to `gnosis-forge`.
tags: [delta, gnosis, plugins, naming, forge]
timestamp: 2026-07-10T12:22:17Z
status: completed
---

# Fulfilled directives

- User selection of `gnosis-forge` as the replacement for `gnosis-bootstrap`.

# Change summary

- Renamed the skill bundle directory to `plugins/gnosis-forge/`.
- Updated Codex, Claude, and Kimi plugin manifests to load the forge skills.
- Documented the vault/forge component boundary and the durable naming decision.

# Verification

- All forge skills pass `quick_validate.py`.
- Codex, Claude, and Kimi manifests parse as JSON and resolve the forge skill path.
- `mise run checks`

# Deviations

None.

# Related decisions

- [Name the knowledge-driven development bundle `gnosis-forge`](../decisions/name-knowledge-driven-development-bundle-gnosis-forge.md)
