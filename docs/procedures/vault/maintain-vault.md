---
type: Procedure
title: maintain-vault
description: Use when auditing or repairing the integrity of a vault.
tags: [gnosis, vault]
invocation: model
---

# maintain-vault

`maintain-vault` repairs high-confidence structural and semantic problems while preserving uncertainty and author-owned meaning decisions.

## Inputs

- Vault configuration, agent rules, and enabled navigation settings.
- Structural validation results and the affected pages.
- Concept Type definitions, linked records, and sources supporting conflicting claims.

## Process

1. Resolve the vault, read its agent rules and configuration, then run `gnosis validate --vault <root>` for the structural baseline.
2. Audit frontmatter, links, orphan pages, near-duplicate identities, stale summaries, and conflicting claims. Audit indexes or logs only when their matching options are enabled.
3. Apply high-confidence repairs in place. Preserve unknown metadata and source-backed disagreements; report identity or meaning conflicts that require author judgment.
4. Run `gnosis index vault --vault <root>` when `vault_index` is enabled and record material repairs only when `vault_log` is enabled.
5. Re-run `gnosis validate --vault <root>` and summarize remaining semantic findings.

## Completion

Structural validation passes and every semantic finding is repaired or reported with its affected paths.
