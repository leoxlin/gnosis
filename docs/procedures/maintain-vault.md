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
- Bounded `audit_knowledge` or `gnosis audit knowledge` findings when available.
- Concept Type definitions, linked records, and sources supporting conflicting claims.

## Process

1. Resolve the vault, read its agent rules and configuration, then run `gnosis validate vault --vault <name>` for the structural baseline.
2. Run `gnosis audit knowledge --vault <name>` or the equivalent read-only `audit_knowledge` tool. Follow every finding cursor. Run staleness separately with an author-appropriate threshold and type or tier filter. If the audit operation is unavailable, inspect the same signals manually.
3. Review each structural fact or semantic candidate and its canonical evidence, using each type's `status`/`tier` lifecycle fields:
   - Authored maintenance: treat `maintenance` entries as explicit judgments, not heuristic candidates. Preserve their authored order, `reason`, `observed_at`, optional `author`, and duplicate `target`. Report broken duplicate targets as structural facts.
   - Orphans: pages with no inbound links that are not type definitions or entry points; rescue by linking from the nearest parent or report.
   - Near-duplicates: pages sharing one identity; merge into the richer page, set `status: archived` plus `superseded_by` on the loser, and repair inbound links.
   - Stale pages: `core`/`supporting` pages whose claims drifted from their sources; refresh or demote `tier` and report.
   - Contradictions: clusters of `^[ambiguous]` markers or conflicting claims; add explicit conflict callouts and report for author judgment.
   - Tag fragmentation: near-identical tags (case, plural, separator variants); normalize to the most-used form.
   - Broken typed `relationships`: invalid targets or relations the Concept Type does not sanction; repair or remove.
4. Apply high-confidence repairs in place through `gnosis apply page`. Never treat a candidate as a proven identity or meaning decision, and never let an audit add, remove, or rewrite `maintenance`. Preserve unknown metadata and source-backed disagreements; report identity or meaning conflicts that require author judgment.
5. Run `gnosis index vault --vault <name>` when `vault_index` is enabled and record material repairs only when `vault_log` is enabled.
6. Re-run `gnosis validate vault --vault <name>` and produce the consolidation report: every finding with its affected paths and the action taken or the author decision needed.

## Completion

Structural validation passes and every consolidation finding is repaired or reported with its affected paths and dispositions.
