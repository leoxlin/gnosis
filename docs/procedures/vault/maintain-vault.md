---
type: Procedure
title: maintain-vault
description: Use when auditing or repairing the integrity of a vault.
tags: [gnosis, vault]
invocation: model
---

# maintain-vault

`maintain-vault` repairs high-confidence structural and semantic problems and consolidates the wiki, while preserving uncertainty and author-owned meaning decisions.

## Inputs

- Vault configuration, agent rules, and enabled navigation settings.
- Structural validation results and the affected pages.
- Concept Type definitions, linked records, and sources supporting conflicting claims.

## Process

1. Resolve the vault, read its agent rules and configuration, then run `gnosis validate vault --vault <root>` for the structural baseline.
2. Audit consolidation findings, using each type's `status`/`tier` lifecycle fields:
   - Orphans: pages with no inbound links that are not type definitions or entry points; rescue by linking from the nearest parent or report.
   - Near-duplicates: pages sharing one identity; merge into the richer page, set `status: archived` plus `superseded_by` on the loser, and repair inbound links.
   - Stale pages: `core`/`supporting` pages whose claims drifted from their sources; refresh or demote `tier` and report.
   - Contradictions: clusters of `^[ambiguous]` markers or conflicting claims; add explicit conflict callouts and report for author judgment.
   - Tag fragmentation: near-identical tags (case, plural, separator variants); normalize to the most-used form.
   - Broken typed `relationships`: invalid targets or relations the Concept Type does not sanction; repair or remove.
3. Apply high-confidence repairs in place through `gnosis apply page`. Preserve unknown metadata and source-backed disagreements; report identity or meaning conflicts that require author judgment.
4. Run `gnosis index vault --vault <root>` when `vault_index` is enabled and record material repairs only when `vault_log` is enabled.
5. Re-run `gnosis validate vault --vault <root>` and produce the consolidation report: every finding with its affected paths and the action taken or the author decision needed.

## Completion

Structural validation passes and every consolidation finding is repaired or reported with its affected paths and dispositions.
